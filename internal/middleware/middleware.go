package middleware

import (
	"log"
	"net/http"

	"hosting/internal/global"
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
		next.ServeHTTP(w, r)
	})
}

// responseWriter 包装 http.ResponseWriter
type responseWriter struct {
	http.ResponseWriter
}
