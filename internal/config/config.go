package config

import (
	"encoding/json"
	"log"
	"os"
	"strconv"

	"hosting/internal/global"
)

func LoadConfig() {
	// 第一步：总是先加载配置文件（提供完整的基础配置）
	file, err := os.ReadFile(global.ConfigFile)
	if err != nil {
		log.Fatalf("Failed to read config file %s: %v", global.ConfigFile, err)
	}

	if err := json.Unmarshal(file, &global.AppConfig); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	log.Println("Configuration loaded from config.json")

	// 第二步：环境变量覆盖特定配置（优先级更高）

	// Telegram配置
	if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
		global.AppConfig.Telegram.Token = token
		log.Println("Telegram token overridden by environment variable")
	}

	if chatID := os.Getenv("TELEGRAM_CHAT_ID"); chatID != "" {
		if id, err := strconv.ParseInt(chatID, 10, 64); err == nil {
			global.AppConfig.Telegram.ChatID = id
			log.Println("Telegram chat ID overridden by environment variable")
		}
	}

	// 数据库配置（新增）
	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		global.AppConfig.Database.Path = dbPath
		log.Printf("Database path overridden by environment variable: %s", dbPath)
	}

	// 服务器端口配置（新增）
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			global.AppConfig.Site.Port = p
			log.Printf("Server port overridden by environment variable: %d", p)
		}
	}

	// 第三步：验证必需配置
	if global.AppConfig.Telegram.Token == "" {
		log.Fatal("Telegram token is not configured. Please set it in config.json or TELEGRAM_BOT_TOKEN environment variable.")
	}

	if global.AppConfig.Database.Path == "" {
		log.Fatal("Database path is not configured. Please check your config.json.")
	}

	log.Printf("Final configuration: database=%s, port=%d",
		global.AppConfig.Database.Path, global.AppConfig.Site.Port)
}
