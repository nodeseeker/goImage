package handlers

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"hosting/internal/db"
	"hosting/internal/global"
	"hosting/internal/r2"
	"hosting/internal/utils"
)

type ImageRecord = global.ImageRecord

type AppError struct {
	Error   error
	Message string
	Code    int
}

func handleError(w http.ResponseWriter, err *AppError) {
	log.Printf("Error: %v", err.Error)
	http.Error(w, err.Message, err.Code)
}

// handleHome 使用 templates/home.html
func HandleHome(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/home.tmpl")
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	// 检查用户登录状态
	isLoggedIn := false
	session, err := global.Store.Get(r, "admin-session")
	if err == nil {
		if auth, ok := session.Values["authenticated"].(bool); ok && auth {
			isLoggedIn = true
		}
	}

	data := struct {
		Title                 string
		Favicon               string
		MaxFileSize           int
		RequireLoginForUpload bool
		IsLoggedIn            bool
	}{
		Title:                 utils.GetPageTitle("图床"),
		Favicon:               global.AppConfig.Site.Favicon,
		MaxFileSize:           global.AppConfig.Site.MaxFileSize,
		RequireLoginForUpload: global.AppConfig.Security.RequireLoginForUpload,
		IsLoggedIn:            isLoggedIn,
	}
	tmpl.Execute(w, data)
}

// HandleUpload 精简说明
func HandleUpload(w http.ResponseWriter, r *http.Request) {
	// 添加请求追踪ID
	requestID := uuid.New().String()
	type contextKey string
	const requestIDKey contextKey = "requestID"
	ctx := context.WithValue(r.Context(), requestIDKey, requestID)

	// 使用defer统一处理panic
	defer func() {
		if err := recover(); err != nil {
			log.Printf("[%s] Panic recovered: %v", requestID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()

	// 使用context控制超时
	ctx, cancel := context.WithTimeout(ctx, global.UploadTimeout)
	defer cancel()
	r = r.WithContext(ctx)

	// 并发控制使用channel代替mutex
	select {
	case global.UploadSemaphore <- struct{}{}:
		defer func() { <-global.UploadSemaphore }()
	default:
		http.Error(w, "Server is busy", http.StatusServiceUnavailable)
		return
	}

	maxSize := int64(global.AppConfig.Site.MaxFileSize * 1024 * 1024)
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	file, header, err := r.FormFile("image")
	if err != nil {
		handleError(w, &AppError{
			Error:   err,
			Message: "无法读取上传文件",
			Code:    http.StatusBadRequest,
		})
		return
	}
	defer file.Close()

	if header.Size > maxSize {
		http.Error(w, "File size exceeds limit", http.StatusBadRequest)
		return
	}

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	file.Seek(0, 0)

	contentType := http.DetectContentType(buffer)
	fileExt, ok := utils.GetFileExtension(contentType)
	if !ok {
		originalExt := utils.NormalizeFileExtension(header.Filename)
		for mime, ext := range global.AllowedMimeTypes {
			if ext == originalExt {
				fileExt = ext
				contentType = mime
				ok = true
				break
			}
		}

		if !ok {
			http.Error(w, "Unsupported file type. Only JPG/JPEG, PNG, GIF and WebP are allowed", http.StatusBadRequest)
			return
		}
	}

	ipAddress := utils.ValidateIPAddress(r.RemoteAddr)
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		ipAddress = utils.ValidateIPAddress(forwardedFor)
	}
	userAgent := utils.SanitizeUserAgent(r.Header.Get("User-Agent"))
	filename := utils.SanitizeFilename(header.Filename)

	tempFile, err := os.CreateTemp("", "upload-*"+fileExt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 生成 R2 对象键名
	proxyUUID := uuid.New().String()
	r2Key := fmt.Sprintf("%s%s", proxyUUID, fileExt)
	proxyURL := fmt.Sprintf("/file/%s%s", proxyUUID, fileExt)

	// 重新打开临时文件用于上传
	tempFile.Seek(0, 0)

	// 上传到 R2
	err = r2.UploadFile(ctx, r2Key, tempFile, contentType)
	if err != nil {
		log.Printf("[%s] Failed to upload to R2: %v", requestID, err)
		http.Error(w, "Failed to upload file", http.StatusInternalServerError)
		return
	}

	var scheme string
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	} else {
		scheme = "http"
	}
	fullURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, proxyURL)

	err = db.WithDBTimeout(func(ctx context.Context) error {
		stmt, err := global.DB.PrepareContext(ctx, `
			INSERT INTO images (
				r2_key, 
				proxy_url, 
				ip_address, 
				user_agent, 
				filename,
				content_type
			) VALUES (?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		_, err = stmt.ExecContext(ctx,
			r2Key,
			proxyURL,
			ipAddress,
			userAgent,
			filename,
			contentType,
		)
		return err
	})

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Error executing statement: %v", err)
		return
	}

	t := template.Must(template.ParseFiles("templates/upload.tmpl"))
	data := struct {
		Title    string
		Favicon  string
		URL      string
		Filename string
	}{
		Title:    utils.GetPageTitle("上传"),
		Favicon:  global.AppConfig.Site.Favicon,
		URL:      fullURL,
		Filename: filename,
	}
	t.Execute(w, data)
}

func HandleImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	// 设置 CORS 头部，允许其他网站嵌入图片
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")

	// 处理 OPTIONS 预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=31536000")
	w.Header().Set("Expires", time.Now().AddDate(1, 0, 0).UTC().Format(http.TimeFormat))

	var r2Key, contentType string
	var isActive bool

	err := db.WithDBTimeout(func(ctx context.Context) error {
		return global.DB.QueryRowContext(ctx, `
            SELECT r2_key, content_type, is_active 
            FROM images 
            WHERE proxy_url LIKE ?`,
			fmt.Sprintf("/file/%s%%", uuid),
		).Scan(&r2Key, &contentType, &isActive)
	})

	if err != nil {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	if !isActive {
		// 尝试读取占位图片
		deletedImage, err := os.ReadFile("static/deleted.jpg")
		if err != nil {
			// 降级处理：占位图片不存在时返回错误
			log.Printf("Failed to read deleted placeholder image: %v", err)
			http.Error(w, "Image has been deleted", http.StatusGone)
			return
		}

		// 设置响应头
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", strconv.Itoa(len(deletedImage)))
		w.Header().Set("Cache-Control", "public, max-age=86400") // 缓存1天
		w.Header().Set("X-Image-Status", "deleted")              // 标识图片状态

		// 返回占位图片
		w.WriteHeader(http.StatusOK)
		w.Write(deletedImage)

		// 记录访问已删除图片的日志
		log.Printf("Served deleted placeholder for UUID: %s", uuid)
		return
	}

	// 更新访问计数
	go func() {
		err := db.WithDBTimeout(func(ctx context.Context) error {
			_, err := global.DB.ExecContext(ctx,
				"UPDATE images SET view_count = view_count + 1 WHERE proxy_url LIKE ?",
				fmt.Sprintf("/file/%s%%", uuid))
			return err
		})
		if err != nil {
			log.Printf("Failed to update view count: %v", err)
		}
	}()

	// 从 R2 获取文件
	rangeHeader := r.Header.Get("Range")
	result, err := r2.GetFileWithRange(r.Context(), r2Key, rangeHeader)
	if err != nil {
		log.Printf("Failed to get file from R2: %v", err)
		http.Error(w, "Failed to retrieve file", http.StatusInternalServerError)
		return
	}
	defer result.Body.Close()

	// 设置响应头
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")

	if result.ContentLength != nil && *result.ContentLength > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", *result.ContentLength))
	}

	// 处理 Range 请求响应
	if result.ContentRange != nil && *result.ContentRange != "" {
		w.Header().Set("Content-Range", *result.ContentRange)
		w.WriteHeader(http.StatusPartialContent)
	}

	// 对于 HEAD 请求，只返回头部信息，不返回文件内容
	if r.Method == "HEAD" {
		return
	}

	// 流式拷贝数据
	buf := make([]byte, 32*1024) // 32KB 缓冲区
	_, err = io.CopyBuffer(w, result.Body, buf)
	if err != nil {
		log.Printf("Error streaming file: %v", err)
	}
}

// 登录页面使用 templates/login.html
func HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	session, err := global.Store.Get(r, "admin-session")
	if err != nil {
		// 清除旧的 session cookie
		cookie := &http.Cookie{
			Name:     "admin-session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
		// 继续处理登录页面，不返回错误
	}

	if auth, ok := session.Values["authenticated"].(bool); ok && auth {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	t := template.Must(template.ParseFiles("templates/login.tmpl"))
	data := struct {
		Title   string
		Favicon string
	}{
		Title:   utils.GetPageTitle("登录"),
		Favicon: global.AppConfig.Site.Favicon,
	}
	t.Execute(w, data)
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	session, err := global.Store.Get(r, "admin-session")
	if err != nil {
		// 清除旧的 session cookie
		cookie := &http.Cookie{
			Name:     "admin-session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
		// 创建新的 session
		session, err = global.Store.New(r, "admin-session")
		if err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}
	}

	username := r.FormValue("username")
	if username == global.AppConfig.Admin.Username && r.FormValue("password") == global.AppConfig.Admin.Password {
		session.Values["authenticated"] = true
		err = session.Save(r, w)
		if err != nil {
			log.Printf("Error saving session: %v", err)
			http.Error(w, "Failed to save session", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	http.Error(w, "Invalid credentials", http.StatusUnauthorized)
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	session, err := global.Store.Get(r, "admin-session")
	if err != nil {
		log.Printf("Error getting session during logout: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	session.Values["authenticated"] = false
	err = session.Save(r, w)
	if err != nil {
		log.Printf("Error saving session during logout: %v", err)
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	//log.Println("User logged out successfully")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// 管理页面使用 templates/admin.html
func HandleAdmin(w http.ResponseWriter, r *http.Request) {
	pageSize := 10
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsedPage, err := strconv.Atoi(p); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}
	offset := (page - 1) * pageSize

	// 获取总记录数
	var total int
	err := global.DB.QueryRow("SELECT COUNT(*) FROM images").Scan(&total)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 获取分页数据
	rows, err := global.DB.Query(`
        SELECT id, proxy_url, ip_address, upload_time, filename, is_active, view_count, content_type
        FROM images 
        ORDER BY upload_time DESC
        LIMIT ? OFFSET ?
    `, pageSize, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var images []ImageRecord
	for rows.Next() {
		var img ImageRecord
		err := rows.Scan(&img.ID, &img.ProxyURL, &img.IPAddress, &img.UploadTime,
			&img.Filename, &img.IsActive, &img.ViewCount, &img.ContentType)
		if err != nil {
			continue
		}
		images = append(images, img)
	}

	totalPages := (total + pageSize - 1) / pageSize

	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"subtract": func(a, b int) int {
			return a - b
		},
	}

	t := template.New("admin.tmpl").Funcs(funcMap)
	t, err = t.ParseFiles("templates/admin.tmpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Title      string
		Favicon    string
		Images     []ImageRecord
		Page       int
		TotalPages int
		HasPrev    bool
		HasNext    bool
	}{
		Title:      utils.GetPageTitle("管理"),
		Favicon:    global.AppConfig.Site.Favicon,
		Images:     images,
		Page:       page,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}
	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func HandleToggleStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := global.DB.Exec("UPDATE images SET is_active = NOT is_active WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
