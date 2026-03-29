package middleware

import (
	"elake-api-gateway/internal/logger"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Logging 日志中间件
func Logging() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.RequestLog.Info("HTTP请求",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}
