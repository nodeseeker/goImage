package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"hosting/internal/db"
	"hosting/internal/global"
	"hosting/internal/template"
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
	tmpl, ok := template.GetTemplate("home")
	if !ok {
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

// HandleUpload 处理图片上传
func HandleUpload(w http.ResponseWriter, r *http.Request) {
	// 添加请求追踪ID用于日志
	requestID := uuid.New().String()

	// 使用defer统一处理panic
	defer func() {
		if err := recover(); err != nil {
			log.Printf("[%s] Panic recovered: %v", requestID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()

	// 使用context控制超时
	ctx, cancel := context.WithTimeout(r.Context(), global.UploadTimeout)
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

	// 根据文件类型选择发送方式
	var message tgbotapi.Message
	var fileID string

	// 对于图片文件（JPG/PNG/WebP），使用 NewPhoto 发送
	// 注意：Telegram 会将动态 WebP 转为静态图片，这是 Telegram 的限制
	if contentType == "image/jpeg" || contentType == "image/jpg" || contentType == "image/png" || contentType == "image/webp" {
		photoMsg := tgbotapi.NewPhoto(global.AppConfig.Telegram.ChatID, tgbotapi.FilePath(tempFile.Name()))
		message, err = global.Bot.Send(photoMsg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// 获取最大尺寸的照片文件ID
		if len(message.Photo) > 0 {
			fileID = message.Photo[len(message.Photo)-1].FileID
		}
	} else {
		// 对于 GIF，使用 Document 方式
		docMsg := tgbotapi.NewDocument(global.AppConfig.Telegram.ChatID, tgbotapi.FilePath(tempFile.Name()))
		message, err = global.Bot.Send(docMsg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fileID = message.Document.FileID
	}
	telegramURL, err := global.Bot.GetFileDirectURL(fileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	proxyUUID := uuid.New().String()
	proxyURL := fmt.Sprintf("/file/%s%s", proxyUUID, fileExt)

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
				telegram_url, 
				proxy_url, 
				ip_address, 
				user_agent, 
				filename,
				content_type,
				file_id
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		_, err = stmt.ExecContext(ctx,
			telegramURL,
			proxyURL,
			ipAddress,
			userAgent,
			filename,
			contentType,
			fileID, // 添加 fileID
		)
		return err
	})

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Error executing statement: %v", err)
		return
	}

	t, ok := template.GetTemplate("upload")
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}
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

func GetTelegramFileURL(fileID string) (string, error) {
	return global.Bot.GetFileDirectURL(fileID)
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

	var telegramURL, contentType string
	var isActive bool
	var fileID string

	err := db.WithDBTimeout(func(ctx context.Context) error {
		return global.DB.QueryRowContext(ctx, `
            SELECT telegram_url, content_type, is_active, file_id 
            FROM images 
            WHERE proxy_url LIKE ?`,
			fmt.Sprintf("/file/%s%%", uuid),
		).Scan(&telegramURL, &contentType, &isActive, &fileID)
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

	// 检查URL缓存
	global.URLCacheMux.RLock()
	cache, exists := global.URLCache[telegramURL]
	global.URLCacheMux.RUnlock()

	var currentURL string
	if !exists || time.Now().After(cache.ExpiresAt) {
		// 获取新的URL
		newURL, err := GetTelegramFileURL(fileID)
		if err != nil {
			http.Error(w, "Failed to refresh file URL", http.StatusInternalServerError)
			return
		}

		// 更新缓存
		global.URLCacheMux.Lock()
		global.URLCache[telegramURL] = &global.FileURLCache{
			URL:       newURL,
			ExpiresAt: time.Now().Add(global.URLCacheTime),
		}
		global.URLCacheMux.Unlock()

		currentURL = newURL

		// 更新数据库中的URL
		err = db.WithDBTimeout(func(ctx context.Context) error {
			tx, err := global.DB.BeginTx(ctx, nil)
			if err != nil {
				return err
			}
			defer func() {
				if err != nil {
					tx.Rollback()
				}
			}()

			// 同时更新 telegram_url 和 view_count
			_, err = tx.ExecContext(ctx,
				"UPDATE images SET telegram_url = ?, view_count = view_count + 1 WHERE proxy_url LIKE ?",
				newURL, fmt.Sprintf("/file/%s%%", uuid))
			if err != nil {
				return err
			}

			return tx.Commit()
		})

		if err != nil {
			log.Printf("Failed to update database: %v", err)
			// 继续处理请求，不返回错误给用户
		}
	} else {
		currentURL = cache.URL

		// 只更新访问计数
		err = db.WithDBTimeout(func(ctx context.Context) error {
			_, err := global.DB.ExecContext(ctx,
				"UPDATE images SET view_count = view_count + 1 WHERE proxy_url LIKE ?",
				fmt.Sprintf("/file/%s%%", uuid))
			return err
		})

		if err != nil {
			log.Printf("Failed to update view count: %v", err)
			// 继续处理请求，不返回错误给用户
		}
	}

	// 创建一个带超时的客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(r.Context(), "GET", currentURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 转发 Range 请求头（支持视频流播放）
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// 动态检测内容类型，特别是处理Telegram转换GIF为MP4的情况
	actualContentType := contentType

	// 只在非Range请求时进行内容检测，避免影响流播放
	isRangeRequest := r.Header.Get("Range") != ""
	needContentDetection := contentType == "image/gif" && !isRangeRequest

	if needContentDetection {
		// 读取前512字节用于内容类型检测
		peekBuffer := make([]byte, 512)
		n, _ := io.ReadAtLeast(resp.Body, peekBuffer, 512)
		if n == 0 {
			// 如果无法读取足够数据，回退到原始长度
			n, _ = resp.Body.Read(peekBuffer)
		}

		// 检测实际内容类型
		detectedType := http.DetectContentType(peekBuffer[:n])

		// 如果检测到是MP4格式，则使用实际的内容类型
		if detectedType == "video/mp4" {
			actualContentType = "video/mp4"
			log.Printf("GIF file converted to MP4 by Telegram, updating content type")
		}

		// 创建包含原始内容的新reader
		resp.Body = io.NopCloser(io.MultiReader(
			bytes.NewReader(peekBuffer[:n]),
			resp.Body,
		))
	} else if contentType == "image/gif" {
		// 对于Range请求，直接假设是MP4（避免破坏流）
		actualContentType = "video/mp4"
	}

	// 设置响应头 - 必须在 WriteHeader 之前设置所有头部
	w.Header().Set("Content-Type", actualContentType)

	// 如果原始响应有内容长度，也设置它
	if resp.ContentLength > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", resp.ContentLength))
	}

	// 设置响应状态码（如果是 Range 请求则为 206）
	if resp.StatusCode == 206 {
		// 转发 Range 相关的响应头 - 在 WriteHeader 之前
		if contentRange := resp.Header.Get("Content-Range"); contentRange != "" {
			w.Header().Set("Content-Range", contentRange)
		}
		if acceptRanges := resp.Header.Get("Accept-Ranges"); acceptRanges != "" {
			w.Header().Set("Accept-Ranges", acceptRanges)
		}
		w.WriteHeader(206)
	} else {
		// 对于普通请求，声明支持 Range 请求
		w.Header().Set("Accept-Ranges", "bytes")
	}

	// 对于 HEAD 请求，只返回头部信息，不返回文件内容
	if r.Method == "HEAD" {
		return
	}

	// 流式拷贝数据
	buf := make([]byte, 32*1024) // 32KB 缓冲区
	_, err = io.CopyBuffer(w, resp.Body, buf)
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

	t, ok := template.GetTemplate("login")
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}
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

	t, ok := template.GetTemplate("admin")
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
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
