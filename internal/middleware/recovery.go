package middleware

import (
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/utils"
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"
)

// Recovery 恢复中间件
func Recovery() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Log.Error("panic recovered",
						zap.Any("error", err),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.ByteString("stack", debug.Stack()),
					)
					utils.Error(w, http.StatusInternalServerError, 5000, "Internal Server Error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
