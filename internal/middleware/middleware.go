package middleware

import (
	"log"
	"net/http"
	"time"

	"hosting/internal/global"
	"hosting/internal/utils"
)

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := global.Store.Get(r, "admin-session")
		if err != nil {
			log.Printf("Error getting session in auth middleware: %v", err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		auth, ok := session.Values["authenticated"].(bool)
		if !ok || !auth {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// LoggingMiddleware 记录HTTP请求日志
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 创建自定义的响应写入器来捕获状态码
		ww := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // 默认状态码
		}

		// 请求处理
		next.ServeHTTP(ww, r)

		// 计算耗时
		duration := time.Since(start)

		// 获取客户端 IP
		clientIP := r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			clientIP = forwardedFor
		}

		// 记录请求信息
		log.Printf(
			"%s - \"%s %s %s\" %d %s %.2fms",
			utils.ValidateIPAddress(clientIP),
			r.Method,
			r.RequestURI,
			r.Proto,
			ww.statusCode,
			r.Header.Get("User-Agent"),
			float64(duration.Microseconds())/1000.0,
		)
	})
}

// responseWriter 包装 http.ResponseWriter
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 重写WriteHeader方法以捕获状态码
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
