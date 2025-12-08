package global

import (
	"database/sql"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gorilla/sessions"
)

var (
	// 全局变量
	DB        *sql.DB
	AppConfig Config
	R2Client  *s3.Client            // R2 客户端
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
)

// Config 应用配置结构
type Config struct {
	R2 struct {
		AccountID       string `json:"accountId"`
		AccessKeyID     string `json:"accessKeyId"`
		AccessKeySecret string `json:"accessKeySecret"`
		BucketName      string `json:"bucketName"`
		PublicURL       string `json:"publicUrl"`
	} `json:"r2"`
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
		RequireLoginForUpload bool     `json:"requireLoginForUpload"` // 是否要求登录才能上传
	} `json:"security"`
}

// ImageRecord 图片记录结构
type ImageRecord struct {
	ID          int
	R2Key       string // R2 对象键名
	ProxyURL    string
	IPAddress   string
	UserAgent   string
	UploadTime  string
	Filename    string
	ContentType string
	IsActive    bool
	ViewCount   int
}
