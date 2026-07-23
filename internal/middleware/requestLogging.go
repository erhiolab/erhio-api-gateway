package middleware

import (
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// RequestLogging 请求日志插件
func RequestLogging() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			start := time.Now()
			wrapped := &responseWriter{ResponseWriter: w, StatusCode: http.StatusOK}
			// 先执行请求
			next.ServeHTTP(wrapped, r)
			node := models.ServiceNode{}
			if selected, ok := ctx.Value(utils.SelectedNodeKey).(*models.SelectedNode); ok && selected != nil && selected.Node != nil {
				node = *selected.Node
			}
			clientIP := models.IPLocation{}
			if ipPtr, ok := ctx.Value(utils.ClientIPKey).(*models.IPLocation); ok && ipPtr != nil {
				clientIP = *ipPtr
			}
			method := r.Method
			path := r.URL.Path
			status := wrapped.StatusCode
			origin := r.Header.Get("Origin")
			userAgent := utils.GetUserAgentInfo(r)
			duration := time.Since(start)
			durationNs := duration.Nanoseconds()
			responseSize := int64(wrapped.Size)
			logger.WithRequestLogCtx(ctx, r).Info("HTTP请求",
				zap.Int64("serviceID", node.ServiceID),
				zap.Int64("nodeID", node.ID),
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
