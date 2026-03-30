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
			wrapped := &ResponseWriter{ResponseWriter: w, StatusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)
			duration := time.Since(start)
			method := r.Method
			path := r.URL.Path
			status := wrapped.StatusCode
			origin := r.Header.Get("Origin")
			durationNs := duration.Nanoseconds()
			responseSize := int64(wrapped.Size)
			logger.RequestLog.Info("HTTP请求",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", status),
				zap.String("origin", origin),
				zap.Int64("durationNs", durationNs),
				zap.Int64("sizeBytes", responseSize),
			)
		})
	}
}
