package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"clamd-api/auth"
)

// LoggingMiddleware 记录请求的中间件
func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	}
}

// AuthMiddleware 使用 API key 进行身份验证的中间件
func AuthMiddleware(next http.HandlerFunc, apiKeyManager *auth.APIKeyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			http.Error(w, "缺少 API key", http.StatusUnauthorized)
			return
		}

		if !apiKeyManager.IsValidAPIKey(apiKey) {
			http.Error(w, "无效的 API key", http.StatusUnauthorized)
			return
		}

		// 获取 API key 的名称，并将其添加到请求上下文中
		if keyName, exists := apiKeyManager.GetAPIKeyName(apiKey); exists {
			r = r.WithContext(context.WithValue(r.Context(), "APIKeyName", keyName))
		}

		next.ServeHTTP(w, r)
	}
}
