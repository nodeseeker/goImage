package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
)

const VERSION = "1.0.0"

func main() {
	var (
		length   = flag.Int("length", 32, "API密钥长度(字节)")
		count    = flag.Int("count", 1, "生成密钥数量")
		showHelp = flag.Bool("help", false, "显示帮助信息")
		showVer  = flag.Bool("version", false, "显示版本信息")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "API密钥生成工具 v%s\n\n", VERSION)
		fmt.Fprintf(os.Stderr, "用法: generate_apikey [选项]\n\n")
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  generate_apikey                    # 生成一个32字节的密钥\n")
		fmt.Fprintf(os.Stderr, "  generate_apikey -count 5           # 生成5个密钥\n")
		fmt.Fprintf(os.Stderr, "  generate_apikey -length 64         # 生成64字节的密钥\n")
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *showVer {
		fmt.Printf("API密钥生成工具 v%s\n", VERSION)
		os.Exit(0)
	}

	if *length < 16 {
		log.Fatal("密钥长度不能小于16字节")
	}

	if *count < 1 {
		log.Fatal("生成数量必须大于0")
	}

	fmt.Printf("生成 %d 个 API 密钥 (长度: %d 字节):\n\n", *count, *length)

	for i := 0; i < *count; i++ {
		key, err := generateAPIKey(*length)
		if err != nil {
			log.Fatalf("生成密钥失败: %v", err)
		}
		fmt.Printf("%d. %s\n", i+1, key)
	}

	fmt.Printf("\n使用说明:\n")
	fmt.Printf("1. 将生成的密钥添加到 config.json 的 security.apiKeys 数组中\n")
	fmt.Printf("2. 设置 security.requireAPIKey 为 true 以启用API认证\n")
	fmt.Printf("3. 客户端使用 -key 参数传递API密钥\n")
}

// generateAPIKey 生成指定长度的随机API密钥
func generateAPIKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
