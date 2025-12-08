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

	// R2配置
	if accountId := os.Getenv("R2_ACCOUNT_ID"); accountId != "" {
		global.AppConfig.R2.AccountID = accountId
		log.Println("R2 account ID overridden by environment variable")
	}

	if accessKeyId := os.Getenv("R2_ACCESS_KEY_ID"); accessKeyId != "" {
		global.AppConfig.R2.AccessKeyID = accessKeyId
		log.Println("R2 access key ID overridden by environment variable")
	}

	if accessKeySecret := os.Getenv("R2_ACCESS_KEY_SECRET"); accessKeySecret != "" {
		global.AppConfig.R2.AccessKeySecret = accessKeySecret
		log.Println("R2 access key secret overridden by environment variable")
	}

	if bucketName := os.Getenv("R2_BUCKET_NAME"); bucketName != "" {
		global.AppConfig.R2.BucketName = bucketName
		log.Println("R2 bucket name overridden by environment variable")
	}

	if publicUrl := os.Getenv("R2_PUBLIC_URL"); publicUrl != "" {
		global.AppConfig.R2.PublicURL = publicUrl
		log.Println("R2 public URL overridden by environment variable")
	}

	// 数据库配置
	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		global.AppConfig.Database.Path = dbPath
		log.Printf("Database path overridden by environment variable: %s", dbPath)
	}

	// 服务器端口配置
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			global.AppConfig.Site.Port = p
			log.Printf("Server port overridden by environment variable: %d", p)
		}
	}

	// 第三步：验证必需配置
	if global.AppConfig.R2.AccountID == "" {
		log.Fatal("R2 account ID is not configured. Please set it in config.json or R2_ACCOUNT_ID environment variable.")
	}

	if global.AppConfig.R2.AccessKeyID == "" {
		log.Fatal("R2 access key ID is not configured. Please set it in config.json or R2_ACCESS_KEY_ID environment variable.")
	}

	if global.AppConfig.R2.AccessKeySecret == "" {
		log.Fatal("R2 access key secret is not configured. Please set it in config.json or R2_ACCESS_KEY_SECRET environment variable.")
	}

	if global.AppConfig.R2.BucketName == "" {
		log.Fatal("R2 bucket name is not configured. Please set it in config.json or R2_BUCKET_NAME environment variable.")
	}

	if global.AppConfig.Database.Path == "" {
		log.Fatal("Database path is not configured. Please check your config.json.")
	}

	log.Printf("Final configuration: database=%s, port=%d",
		global.AppConfig.Database.Path, global.AppConfig.Site.Port)
}
