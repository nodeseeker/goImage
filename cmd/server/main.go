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
	"hosting/internal/middleware"
	"hosting/internal/telegram"
)

func main() {
	// 加载配置
	config.LoadConfig()

	// 初始化数据库
	db.InitDB()

	// 初始化 Telegram bot
	telegram.InitTelegram()

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

	r := mux.NewRouter()

	// 静态文件
	fs := http.FileServer(http.Dir(global.StaticDir))
	r.PathPrefix("/favicon.ico").Handler(fs)
	r.PathPrefix("/robots.txt").Handler(fs)

	// 路由设置
	r.HandleFunc("/", handlers.HandleHome).Methods("GET")
	r.HandleFunc("/upload", handlers.HandleUpload).Methods("POST")
	r.HandleFunc("/file/{uuid}", handlers.HandleImage).Methods("GET")
	r.HandleFunc("/login", handlers.HandleLoginPage).Methods("GET")
	r.HandleFunc("/login", handlers.HandleLogin).Methods("POST")
	r.HandleFunc("/logout", handlers.HandleLogout).Methods("GET")
	r.HandleFunc("/admin", middleware.RequireAuth(handlers.HandleAdmin)).Methods("GET")
	r.HandleFunc("/admin/toggle/{id}", middleware.RequireAuth(handlers.HandleToggleStatus)).Methods("POST")

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
		Addr:         addr,
		Handler:      r, // 移除 LoggingMiddleware
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
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
	<-stop

	// 优雅关闭时增加超时控制
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	if err := global.DB.Close(); err != nil {
		log.Printf("Database close error: %v", err)
	}
}
