package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"

	"hosting/internal/db"
	"hosting/internal/global"
	"hosting/internal/logger"
	"hosting/internal/utils"
)

// APIResponse 定义通用API响应结构
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ImageResponse 包含上传后的图片信息
type ImageResponse struct {
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	UploadTime  string `json:"uploadTime"`
}

// HandleAPIUpload 处理通过API上传图片
func HandleAPIUpload(w http.ResponseWriter, r *http.Request) {
	// 设置响应类型为JSON
	w.Header().Set("Content-Type", "application/json")

	// 处理跨域请求
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 添加请求追踪ID
	requestID := uuid.New().String()
	logger.Debug("开始处理API上传请求: %s", requestID)

	// 使用defer统一处理panic
	defer func() {
		if err := recover(); err != nil {
			logger.Error("[%s] API上传处理中发生panic: %v", requestID, err)
			sendJSONError(w, "服务器内部错误", http.StatusInternalServerError)
		}
	}()

	// 使用context控制超时
	ctx, cancel := context.WithTimeout(r.Context(), global.UploadTimeout)
	defer cancel()
	r = r.WithContext(ctx)

	// 并发控制
	select {
	case global.UploadSemaphore <- struct{}{}:
		defer func() { <-global.UploadSemaphore }()
	default:
		sendJSONError(w, "服务器繁忙，请稍后再试", http.StatusServiceUnavailable)
		return
	}

	// 限制上传文件大小
	maxSize := int64(global.AppConfig.Site.MaxFileSize * 1024 * 1024)
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	// 解析多部分表单
	err := r.ParseMultipartForm(maxSize)
	if err != nil {
		sendJSONError(w, "无法解析表单数据", http.StatusBadRequest)
		return
	}

	// 获取上传文件
	file, header, err := r.FormFile("image")
	if err != nil {
		sendJSONError(w, "无法读取上传文件", http.StatusBadRequest)
		return
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			logger.Error("[%s] failed to close uploaded file: %v", requestID, cerr)
		}
	}()

	// 检查文件大小
	if header.Size > maxSize {
		sendJSONError(w, fmt.Sprintf("文件大小超过限制 (%dMB)", global.AppConfig.Site.MaxFileSize), http.StatusBadRequest)
		return
	}

	// 检查文件类型
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		sendJSONError(w, "读取文件内容失败", http.StatusInternalServerError)
		return
	}
	if _, err = file.Seek(0, 0); err != nil {
		sendJSONError(w, "读取文件失败", http.StatusInternalServerError)
		return
	}

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
			sendJSONError(w, "不支持的文件类型，仅支持JPG/JPEG, PNG, GIF和WebP格式", http.StatusBadRequest)
			return
		}
	}

	// 记录客户端信息
	ipAddress := utils.ValidateIPAddress(r.RemoteAddr)
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		ipAddress = utils.ValidateIPAddress(forwardedFor)
	}
	userAgent := utils.SanitizeUserAgent(r.Header.Get("User-Agent"))
	filename := utils.SanitizeFilename(header.Filename)

	// 创建临时文件
	tempFile, err := os.CreateTemp("", "upload-*"+fileExt)
	if err != nil {
		sendJSONError(w, "创建临时文件失败", http.StatusInternalServerError)
		return
	}
	defer func() {
		if cerr := os.Remove(tempFile.Name()); cerr != nil {
			logger.Error("[%s] failed to remove temp file %s: %v", requestID, tempFile.Name(), cerr)
		}
	}()
	defer func() {
		if cerr := tempFile.Close(); cerr != nil {
			logger.Error("[%s] failed to close temp file %s: %v", requestID, tempFile.Name(), cerr)
		}
	}()

	// 复制上传文件到临时文件
	_, err = io.Copy(tempFile, file)
	if err != nil {
		sendJSONError(w, "保存上传文件失败", http.StatusInternalServerError)
		return
	}

	// 根据文件类型选择发送方式
	var message tgbotapi.Message
	var fileID string

	// 对于图片文件，使用NewPhoto发送以确保在Telegram中正确显示
	if contentType == "image/jpeg" || contentType == "image/jpg" || contentType == "image/png" || contentType == "image/webp" {
		photoMsg := tgbotapi.NewPhoto(global.AppConfig.Telegram.ChatID, tgbotapi.FilePath(tempFile.Name()))
		message, err = global.Bot.Send(photoMsg)
		if err != nil {
			sendJSONError(w, "上传到存储服务失败", http.StatusInternalServerError)
			return
		}
		// 获取最大尺寸的照片文件ID
		if len(message.Photo) > 0 {
			fileID = message.Photo[len(message.Photo)-1].FileID
		}
	} else {
		// 对于GIF等其他格式，仍使用Document方式
		docMsg := tgbotapi.NewDocument(global.AppConfig.Telegram.ChatID, tgbotapi.FilePath(tempFile.Name()))
		message, err = global.Bot.Send(docMsg)
		if err != nil {
			sendJSONError(w, "上传到存储服务失败", http.StatusInternalServerError)
			return
		}
		fileID = message.Document.FileID
	}
	telegramURL, err := global.Bot.GetFileDirectURL(fileID)
	if err != nil {
		sendJSONError(w, "获取文件URL失败", http.StatusInternalServerError)
		return
	}

	// 生成公开URL
	proxyUUID := uuid.New().String()
	proxyURL := fmt.Sprintf("/file/%s%s", proxyUUID, fileExt)

	// 构建完整URL
	var scheme string
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	} else {
		scheme = "http"
	}
	fullURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, proxyURL)

	// 存储记录到数据库
	uploadTime := time.Now().Format(time.RFC3339)
	err = db.WithDBTimeout(func(ctx context.Context) error {
		stmt, err := global.DB.PrepareContext(ctx, `
			INSERT INTO images (
				telegram_url, 
				proxy_url, 
				ip_address, 
				user_agent, 
				filename,
				content_type,
				file_id,
				upload_time
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer func() {
			if cerr := stmt.Close(); cerr != nil {
				logger.Error("[%s] failed to close statement: %v", requestID, cerr)
			}
		}()

		_, err = stmt.ExecContext(ctx,
			telegramURL,
			proxyURL,
			ipAddress,
			userAgent,
			filename,
			contentType,
			fileID,
			uploadTime,
		)
		return err
	})

	if err != nil {
		logger.Error("数据库插入失败: %v", err)
		sendJSONError(w, "保存记录失败", http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	imageResponse := ImageResponse{
		URL:         fullURL,
		Filename:    filename,
		ContentType: contentType,
		Size:        header.Size,
		UploadTime:  uploadTime,
	}

	response := APIResponse{
		Success: true,
		Message: "上传成功",
		Data:    imageResponse,
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("[%s] failed to write JSON response: %v", requestID, err)
	}
}

// sendJSONError 发送JSON格式的错误响应
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	response := APIResponse{
		Success: false,
		Message: message,
	}

	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("failed to write JSON error response: %v", err)
	}
}

// HandleAPIHealthCheck 健康检查API
func HandleAPIHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := APIResponse{
		Success: true,
		Message: "API服务正常",
		Data: map[string]any{
			"version":   "1.0",
			"status":    "operational",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("failed to write health check response: %v", err)
	}
}
