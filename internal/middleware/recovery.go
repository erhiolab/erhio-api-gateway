package middleware

import (
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/utils"
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"
)

// Recovery 服务恢复插件
func Recovery() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.WithRequestLogCtx(r.Context(), r).Error("服务恢复插件: panic recovered",
						zap.Any("error", err),
						zap.ByteString("stack", debug.Stack()),
					)
					utils.InternalServerError(w)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
