package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"hosting/internal/global"
)

// StatusData 系统状态信息
type StatusData struct {
	Status       string    `json:"status"`
	StartTime    time.Time `json:"startTime"`
	Uptime       string    `json:"uptime"`
	GoVersion    string    `json:"goVersion"`
	NumGoroutine int       `json:"numGoroutine"`
	NumCPU       int       `json:"numCPU"`
	MemStats     struct {
		Alloc        uint64 `json:"alloc"`        // 当前分配的内存
		TotalAlloc   uint64 `json:"totalAlloc"`   // 累计分配的内存
		Sys          uint64 `json:"sys"`          // 系统分配的内存
		NumGC        uint32 `json:"numGC"`        // GC运行次数
		PauseTotalNs uint64 `json:"pauseTotalNs"` // GC暂停总时间
	} `json:"memStats"`
	URLCacheSize int `json:"urlCacheSize"` // URL缓存数量
}

var (
	startTime = time.Now()
)

// HandleStatus 返回应用状态信息
func HandleStatus(w http.ResponseWriter, r *http.Request) {
	// 检查身份验证
	if r.URL.Query().Get("key") != global.AppConfig.Security.StatusKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 获取URL缓存大小
	global.URLCacheMux.RLock()
	urlCacheSize := len(global.URLCache)
	global.URLCacheMux.RUnlock()

	status := StatusData{
		Status:       "ok",
		StartTime:    startTime,
		Uptime:       time.Since(startTime).String(),
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
		URLCacheSize: urlCacheSize,
	}

	status.MemStats.Alloc = memStats.Alloc
	status.MemStats.TotalAlloc = memStats.TotalAlloc
	status.MemStats.Sys = memStats.Sys
	status.MemStats.NumGC = memStats.NumGC
	status.MemStats.PauseTotalNs = memStats.PauseTotalNs

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// HandleHealthCheck 健康检查接口
func HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// 检查数据库连接
	if err := global.DB.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Database connection failed",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Service is healthy",
	})
}
