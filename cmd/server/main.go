package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"hosting/internal/config"
	"hosting/internal/db"
	"hosting/internal/global"
	"hosting/internal/handlers"
	"hosting/internal/logger"
	"hosting/internal/middleware"
	"hosting/internal/telegram"
	"hosting/internal/template"
)

func main() {
	// 初始化日志系统
	if os.Getenv("DEBUG") == "true" {
		logger.InitLogger(logger.DebugLevel)
	} else {
		logger.InitLogger(logger.InfoLevel)
	}
	logger.Info("图床服务启动中...")

	// 加载配置
	config.LoadConfig()
	logger.Info("配置加载完成")

	// 初始化数据库
	db.InitDB()
	logger.Info("数据库连接初始化完成")

	// 初始化 Telegram bot
	telegram.InitTelegram()
	logger.Info("Telegram 机器人初始化完成")

	// 初始化模板
	template.InitTemplates()
	logger.Info("模板初始化完成")

	// 生成随机 session secret
	var sessionSecret []byte
	if global.AppConfig.Security.SessionSecret != "" {
		sessionSecret = []byte(global.AppConfig.Security.SessionSecret)
	} else {
		sessionSecret = make([]byte, 32)
		if _, err := rand.Read(sessionSecret); err != nil {
			log.Fatal("Failed to generate session secret:", err)
		}
		// 将生成的密钥保存到配置中
		global.AppConfig.Security.SessionSecret = base64.StdEncoding.EncodeToString(sessionSecret)
	}

	// 根据环境配置设置开发模式
	global.IsDevelopment = global.AppConfig.Environment == "development"

	// 修改 session 配置以支持开发环境
	global.Store = sessions.NewCookieStore(sessionSecret)
	global.Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   !global.IsDevelopment, // 在开发环境下允许 HTTP
		SameSite: http.SameSiteStrictMode,
	}

	// 确保静态文件目录存在
	if _, err := os.Stat(global.StaticDir); os.IsNotExist(err) {
		err = os.MkdirAll(global.StaticDir, 0755)
		if err != nil {
			log.Fatal("Failed to create static directory:", err)
		}
	}

	// 创建全局上传信号量
	global.UploadSemaphore = make(chan struct{}, global.MaxConcurrentUploads)

	// 启动缓存清理定时器
	go func() {
		cacheTicker := time.NewTicker(12 * time.Hour)
		defer cacheTicker.Stop()

		for range cacheTicker.C {
			cleanURLCache()
		}
	}()

	r := mux.NewRouter()

	// 静态文件
	fs := http.FileServer(http.Dir(global.StaticDir))
	r.PathPrefix("/favicon.ico").Handler(fs)
	r.PathPrefix("/robots.txt").Handler(fs)

	// 路由设置
	r.HandleFunc("/", handlers.HandleHome).Methods("GET")
	r.HandleFunc("/upload", middleware.RequireAuthForUpload(handlers.HandleUpload)).Methods("POST")
	r.HandleFunc("/file/{uuid}", handlers.HandleImage).Methods("GET", "HEAD")
	r.HandleFunc("/login", handlers.HandleLoginPage).Methods("GET")
	r.HandleFunc("/login", handlers.HandleLogin).Methods("POST")
	r.HandleFunc("/logout", handlers.HandleLogout).Methods("GET")
	r.HandleFunc("/admin", middleware.RequireAuth(handlers.HandleAdmin)).Methods("GET")
	r.HandleFunc("/admin/toggle/{id}", middleware.RequireAuth(handlers.HandleToggleStatus)).Methods("POST")

	// RESTful API 路由
	apiRouter := r.PathPrefix("/api/v1").Subrouter()
	apiRouter.HandleFunc("/upload", middleware.RequireAPIKey(handlers.HandleAPIUpload)).Methods("POST", "OPTIONS")
	apiRouter.HandleFunc("/health", handlers.HandleAPIHealthCheck).Methods("GET")

	// 状态监控和健康检查路由
	r.HandleFunc("/health", handlers.HandleHealthCheck).Methods("GET")
	r.HandleFunc("/status", handlers.HandleStatus).Methods("GET")

	// 服务器配置
	port := global.AppConfig.Site.Port
	if port == 0 {
		port = 8080
	}
	host := global.AppConfig.Site.Host
	if host == "" {
		host = "127.0.0.1"
	}
	addr := fmt.Sprintf("%s:%d", host, port)

	// 添加优雅关闭超时配置
	shutdownTimeout := 30 * time.Second

	srv := &http.Server{
		Addr:           addr,
		Handler:        middleware.LoggingMiddleware(r), // 添加日志记录中间件
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   30 * time.Second,  // 增加写入超时，处理大文件
		IdleTimeout:    120 * time.Second, // 增加空闲连接超时
		MaxHeaderBytes: 1 << 20,           // 限制请求头大小为 1MB
	}

	// 启动服务器
	go func() {
		log.Printf("Server is running on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// 优雅关闭
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// 等待终止信号
	sig := <-stop
	logger.Info("收到终止信号: %v，开始优雅关闭服务...", sig)

	// 优雅关闭时增加超时控制
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	logger.Info("正在关闭 HTTP 服务器...")
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("HTTP 服务器关闭错误: %v", err)
	} else {
		logger.Info("HTTP 服务器关闭成功")
	}

	logger.Info("正在关闭数据库连接...")
	if err := global.DB.Close(); err != nil {
		logger.Error("数据库关闭错误: %v", err)
	} else {
		logger.Info("数据库连接关闭成功")
	}

	logger.Info("服务已完全关闭，感谢使用")
}

// cleanURLCache 清理过期的URL缓存，防止内存泄漏
func cleanURLCache() {
	logger.Info("开始清理URL缓存...")
	count := 0

	global.URLCacheMux.Lock()
	defer global.URLCacheMux.Unlock()

	now := time.Now()
	for key, cache := range global.URLCache {
		if now.After(cache.ExpiresAt) {
			delete(global.URLCache, key)
			count++
		}
	}

	logger.Info("URL缓存清理完成，共清理 %d 个过期项", count)
}
