package global

import (
	"database/sql"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gorilla/sessions"
)

var (
	// 全局变量
	DB        *sql.DB
	AppConfig Config
	Bot       *tgbotapi.BotAPI
	Store     *sessions.CookieStore // 移除初始化，将在 main 中进行

	// 并发控制
	UploadSemaphore chan struct{} // 用于限制并发上传

	// 程序配置
	ConfigFile           = "./config.json"
	StaticDir            = "./static"
	MaxConcurrentUploads = 5
	DBTimeout            = 10 * time.Second
	UploadTimeout        = 30 * time.Second

	// 允许的文件类型
	AllowedMimeTypes = map[string]string{
		"image/jpeg": ".jpg",
		"image/jpg":  ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
		"image/webp": ".webp",
	}

	IsDevelopment = false // 开发环境标志，默认为生产环境

	// URLCache 用于存储文件URL的缓存
	URLCache     = make(map[string]*FileURLCache)
	URLCacheMux  sync.RWMutex
	URLCacheTime = 23 * time.Hour // Telegram URL 通常 24 小时过期
)

// Config 应用配置结构
type Config struct {
	Telegram struct {
		Token  string `json:"token"`
		ChatID int64  `json:"chatId"`
	} `json:"telegram"`
	Admin struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"admin"`
	Database struct {
		Path            string `json:"path"`
		MaxOpenConns    int    `json:"maxOpenConns"`
		MaxIdleConns    int    `json:"maxIdleConns"`
		ConnMaxLifetime string `json:"connMaxLifetime"`
	} `json:"database"`
	Site struct {
		Name        string `json:"name"`
		Favicon     string `json:"favicon"`
		MaxFileSize int    `json:"maxFileSize"`
		Port        int    `json:"port"`
		Host        string `json:"host"`
	} `json:"site"`
	Security struct {
		RateLimit struct {
			Enabled bool   `json:"enabled"`
			Limit   int    `json:"limit"`
			Window  string `json:"window"`
		} `json:"rateLimit"`
		AllowedHosts          []string `json:"allowedHosts"`
		SessionSecret         string   `json:"sessionSecret"`         // session secret 配置
		StatusKey             string   `json:"statusKey"`             // 状态页面访问密钥
		APIKeys               []string `json:"apiKeys"`               // API 密钥列表
		RequireAPIKey         bool     `json:"requireAPIKey"`         // 是否强制要求 API Key
		RequireLoginForUpload bool     `json:"requireLoginForUpload"` // 是否要求登录才能上传
	} `json:"security"`
	Environment string `json:"environment"` // 可选值: "development" 或 "production"
}

// ImageRecord 图片记录结构
type ImageRecord struct {
	ID          int
	TelegramURL string
	ProxyURL    string
	IPAddress   string
	UserAgent   string
	UploadTime  string
	Filename    string
	ContentType string
	IsActive    bool
	ViewCount   int
}

// FileURLCache 用于缓存文件URL
type FileURLCache struct {
	URL       string
	ExpiresAt time.Time
}
