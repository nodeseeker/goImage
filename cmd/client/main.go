package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// 响应结构体
type APIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type ImageResponse struct {
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	UploadTime  string `json:"uploadTime"`
}

// 设置版本信息
const (
	VERSION = "0.1.4"
)

func main() {
	// 定义命令行参数
	var (
		url       = flag.String("url", "", "图床服务器API地址 (例如: https://img.example.com/api/v1/upload)")
		filePath  = flag.String("file", "", "要上传的图片文件路径")
		apiKey    = flag.String("key", "", "API密钥 (如果服务器需要认证)")
		timeout   = flag.Int("timeout", 60, "上传超时时间(秒)")
		showHelp  = flag.Bool("help", false, "显示帮助信息")
		showVer   = flag.Bool("version", false, "显示版本信息")
		verbosity = flag.Bool("verbose", false, "显示详细输出")
	)

	// 自定义usage信息
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "GoImage 命令行客户端 v%s\n\n", VERSION)
		fmt.Fprintf(os.Stderr, "用法: client -url URL -file 文件路径 [选项]\n\n")
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  client -url https://img.example.com/api/v1/upload -file ./image.jpg\n")
		fmt.Fprintf(os.Stderr, "  client -url https://img.example.com/api/v1/upload -file ./image.jpg -key your-api-key\n")
	}

	// 解析命令行参数
	flag.Parse()

	// 显示帮助信息
	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	// 显示版本信息
	if *showVer {
		fmt.Printf("GoImage 命令行客户端 v%s\n", VERSION)
		os.Exit(0)
	}

	// 检查必要参数
	if *url == "" {
		log.Fatal("错误: 必须指定服务器API地址，使用 -url 参数")
	}

	if *filePath == "" {
		log.Fatal("错误: 必须指定图片文件路径，使用 -file 参数")
	}

	// 检查文件是否存在
	if _, err := os.Stat(*filePath); os.IsNotExist(err) {
		log.Fatalf("错误: 图片文件不存在: %s", *filePath)
	}

	// 验证文件类型
	ext := strings.ToLower(filepath.Ext(*filePath))
	isValidExt := slices.Contains([]string{".jpg", ".jpeg", ".png", ".gif", ".webp"}, ext)

	if !isValidExt {
		log.Fatalf("错误: 不支持的文件类型: %s，仅支持JPG/JPEG, PNG, GIF和WebP格式", ext)
	}

	// 检查URL格式
	if !strings.HasPrefix(*url, "http://") && !strings.HasPrefix(*url, "https://") {
		*url = "http://" + *url
		if *verbosity {
			log.Printf("已为URL添加http://前缀: %s", *url)
		}
	}

	// 上传图片
	result, err := uploadImage(*url, *filePath, *apiKey, *timeout, *verbosity)
	if err != nil {
		log.Fatalf("上传失败: %v", err)
	}

	// 输出结果
	fmt.Printf("上传成功! 图片URL: %s\n", result.URL)

	if *verbosity {
		fmt.Printf("文件名: %s\n", result.Filename)
		fmt.Printf("类型: %s\n", result.ContentType)
		fmt.Printf("大小: %.2f KB\n", float64(result.Size)/1024)
		fmt.Printf("上传时间: %s\n", result.UploadTime)
	}
}

// uploadImage 上传图片到服务器
func uploadImage(serverURL, imagePath, apiKey string, timeoutSeconds int, verbose bool) (*ImageResponse, error) {
	if verbose {
		log.Printf("准备上传文件: %s 到 %s", imagePath, serverURL)
		if apiKey != "" {
			log.Printf("使用API密钥认证")
		}
	}

	// 打开文件
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件: %v", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			log.Printf("warning: failed to close file %s: %v", imagePath, cerr)
		}
	}()

	// 创建multipart表单
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// 创建文件字段
	fileField, err := writer.CreateFormFile("image", filepath.Base(imagePath))
	if err != nil {
		return nil, fmt.Errorf("创建表单字段失败: %v", err)
	}

	// 复制文件内容到表单
	if verbose {
		log.Printf("正在读取文件内容...")
	}

	_, err = io.Copy(fileField, file)
	if err != nil {
		return nil, fmt.Errorf("写入文件内容失败: %v", err)
	}

	// 关闭writer以完成表单
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("关闭表单失败: %v", err)
	}

	// 创建HTTP请求
	if verbose {
		log.Printf("创建HTTP请求...")
	}

	req, err := http.NewRequest("POST", serverURL, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", fmt.Sprintf("GoImage-Client/%s", VERSION))

	// 如果提供了 API Key，添加到请求头
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	// 发送请求
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}

	if verbose {
		log.Printf("发送请求中，超时设置为 %d 秒...", timeoutSeconds)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("warning: failed to close response body: %v", cerr)
		}
	}()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if verbose {
		log.Printf("收到响应，状态码: %d", resp.StatusCode)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务器返回错误: %s, 状态码: %d", string(body), resp.StatusCode)
	}

	// 解析JSON响应
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("上传失败: %s", apiResp.Message)
	}

	// 解析图片数据
	var imgResp ImageResponse
	if err := json.Unmarshal(apiResp.Data, &imgResp); err != nil {
		return nil, fmt.Errorf("解析图片数据失败: %v", err)
	}

	return &imgResp, nil
}
