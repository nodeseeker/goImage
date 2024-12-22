package utils

import (
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"hosting/internal/global"
)

func GetFileExtension(mimeType string) (string, bool) {
	ext, ok := global.AllowedMimeTypes[mimeType]
	return ext, ok
}

func NormalizeFileExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".jpeg" {
		return ".jpg"
	}
	return ext
}

func ValidateIPAddress(ip string) string {
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	if net.ParseIP(ip) == nil {
		return "unknown"
	}
	return ip
}

func SanitizeUserAgent(ua string) string {
	reg := regexp.MustCompile(`[^\w\s\-\.,;:/\(\)]`)
	ua = reg.ReplaceAllString(ua, "")
	if len(ua) > 255 {
		ua = ua[:255]
	}
	return ua
}

func SanitizeFilename(filename string) string {
	filename = filepath.Base(filename)
	reg := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	filename = reg.ReplaceAllString(filename, "")
	if filename == "" || filename == "." {
		now := time.Now()
		filename = fmt.Sprintf("image_%s", now.Format("20060102150405"))
	}
	if len([]rune(filename)) > 100 {
		runes := []rune(filename)
		filename = string(runes[:100])
	}
	return filename
}

func GetPageTitle(page string) string {
	return fmt.Sprintf("%s | %s", page, global.AppConfig.Site.Name)
}
