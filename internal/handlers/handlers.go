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

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"hosting/internal/db"
	"hosting/internal/global"
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
	t := template.Must(template.ParseFiles("templates/home.tmpl"))
	data := struct {
		Title       string
		Favicon     string
		MaxFileSize int
	}{
		Title:       utils.GetPageTitle("图床"),
		Favicon:     global.AppConfig.Site.Favicon,
		MaxFileSize: global.AppConfig.Site.MaxFileSize,
	}
	t.Execute(w, data)
}

// HandleUpload 精简说明
func HandleUpload(w http.ResponseWriter, r *http.Request) {
	// 添加请求追踪ID
	requestID := uuid.New().String()
	ctx := context.WithValue(r.Context(), "requestID", requestID)

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

	msg := tgbotapi.NewDocument(global.AppConfig.Telegram.ChatID, tgbotapi.FilePath(tempFile.Name()))
	message, err := global.Bot.Send(msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileID := message.Document.FileID
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
				content_type
			) VALUES (?, ?, ?, ?, ?, ?)
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

	w.Header().Set("Cache-Control", "public, max-age=31536000")
	w.Header().Set("Expires", time.Now().AddDate(1, 0, 0).UTC().Format(http.TimeFormat))

	var telegramURL, contentType string
	var isActive bool

	err := db.WithDBTimeout(func(ctx context.Context) error {
		return global.DB.QueryRowContext(ctx, `
            SELECT telegram_url, content_type, is_active 
            FROM images 
            WHERE proxy_url LIKE ?`,
			fmt.Sprintf("/file/%s%%", uuid),
		).Scan(&telegramURL, &contentType, &isActive)
	})

	if err != nil {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	if !isActive {
		http.Error(w, "Image has been deleted", http.StatusGone)
		return
	}

	_, err = global.DB.Exec("UPDATE images SET view_count = view_count + 1 WHERE proxy_url LIKE ?",
		fmt.Sprintf("/file/%s%%", uuid))
	if err != nil {
		log.Printf("Failed to update view count: %v", err)
	}

	resp, err := http.Get(telegramURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", contentType)
	io.Copy(w, resp.Body)
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
        SELECT id, proxy_url, ip_address, upload_time, filename, is_active, view_count
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
			&img.Filename, &img.IsActive, &img.ViewCount)
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
