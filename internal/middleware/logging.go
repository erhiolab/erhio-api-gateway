package middleware

import (
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Logging 日志插件
func Logging() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &ResponseWriter{ResponseWriter: w, StatusCode: http.StatusOK}
			// 先执行请求
			next.ServeHTTP(wrapped, r)
			node, ok := r.Context().Value(utils.SelectedNodeKey).(*models.ServiceNode)
			if !ok {
				utils.BadGateway(w)
				return
			}
			requestID := "unknown"
			if id, ok := r.Context().Value(utils.RequestIDKey).(string); ok {
				requestID = id
			}
			clientIP := models.IPLocation{}
			if ipPtr, ok := r.Context().Value(utils.ClientIPKey).(*models.IPLocation); ok && ipPtr != nil {
				clientIP = *ipPtr
			}
			method := r.Method
			path := r.URL.Path
			status := wrapped.StatusCode
			origin := r.Header.Get("Origin")
			userAgent := models.UserAgent{}
			if uaPtr, ok := r.Context().Value(utils.UserAgentKey).(*models.UserAgent); ok && uaPtr != nil {
				userAgent = *uaPtr
			}
			duration := time.Since(start)
			durationNs := duration.Nanoseconds()
			responseSize := int64(wrapped.Size)
			logger.RequestLog.Info("HTTP请求",
				zap.Int64("serviceID", node.ServiceID),
				zap.Int64("nodeID", node.ID),
				zap.String("requestID", requestID),
				zap.Any("clientIP", clientIP),
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", status),
				zap.String("origin", origin),
				zap.Any("userAgent", userAgent),
				zap.Int64("durationNs", durationNs),
				zap.Int64("sizeBytes", responseSize),
			)
		})
	}
}
