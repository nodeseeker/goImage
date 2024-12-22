package db

import (
	"context"
	"database/sql"
	"log"
	"time"

	"hosting/internal/global"

	_ "modernc.org/sqlite"
)

func InitDB() {
	dbPath := "./images.db"
	if global.AppConfig.Database.Path != "" {
		dbPath = global.AppConfig.Database.Path
	}

	var err error
	global.DB, err = sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	_, err = global.DB.Exec(`
	CREATE TABLE IF NOT EXISTS images (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_url TEXT NOT NULL,
		proxy_url TEXT NOT NULL,
		ip_address TEXT NOT NULL,
		user_agent TEXT NOT NULL,
		upload_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		filename TEXT NOT NULL,
		content_type TEXT NOT NULL,
		is_active BOOLEAN DEFAULT 1,
		view_count INTEGER DEFAULT 0
	)`)

	if err != nil {
		log.Fatal(err)
	}

	_, err = global.DB.Exec(`
    CREATE INDEX IF NOT EXISTS idx_proxy_url ON images(proxy_url);
    CREATE INDEX IF NOT EXISTS idx_upload_time ON images(upload_time);
    CREATE INDEX IF NOT EXISTS idx_is_active ON images(is_active);
    `)

	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}

	// 设置数据库连接池参数
	maxOpenConns := 25
	if global.AppConfig.Database.MaxOpenConns > 0 {
		maxOpenConns = global.AppConfig.Database.MaxOpenConns
	}
	global.DB.SetMaxOpenConns(maxOpenConns)

	maxIdleConns := 10
	if global.AppConfig.Database.MaxIdleConns > 0 {
		maxIdleConns = global.AppConfig.Database.MaxIdleConns
	}
	global.DB.SetMaxIdleConns(maxIdleConns)

	connMaxLifetime := 5 * time.Minute
	if global.AppConfig.Database.ConnMaxLifetime != "" {
		if d, err := time.ParseDuration(global.AppConfig.Database.ConnMaxLifetime); err == nil {
			connMaxLifetime = d
		}
	}
	global.DB.SetConnMaxLifetime(connMaxLifetime)

	if err := global.DB.Ping(); err != nil {
		log.Fatal("Database connection failed:", err)
	}
}

// 数据库操作超时包装函数
func WithDBTimeout(f func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), global.DBTimeout)
	defer cancel()
	return f(ctx)
}
