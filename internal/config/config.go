package config

import (
	"encoding/json"
	"log"
	"os"
	"strconv"

	"hosting/internal/global"
)

func LoadConfig() {
	// 优先使用环境变量
	if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
		global.AppConfig.Telegram.Token = token
	}

	if chatID := os.Getenv("TELEGRAM_CHAT_ID"); chatID != "" {
		if id, err := strconv.ParseInt(chatID, 10, 64); err == nil {
			global.AppConfig.Telegram.ChatID = id
		}
	}

	// 如果环境变量未设置，回退到配置文件
	if global.AppConfig.Telegram.Token == "" {
		file, err := os.ReadFile(global.ConfigFile)
		if err != nil {
			log.Fatal(err)
		}

		if err := json.Unmarshal(file, &global.AppConfig); err != nil { // 更新引用
			log.Fatal(err)
		}
	}
}
