package middleware

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/utils"
	"net/http"

	"go.uber.org/zap"
)

// ConcurrencyLimit 并发限制插件
func ConcurrencyLimit(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			// 尝试获取并发许可
			if !app.ConcurrencyLimiter.Acquire() {
				logger.WithRequestLogCtx(ctx).Warn("并发数超出限制",
					zap.Int32("current", app.ConcurrencyLimiter.GetCurrentConcurrency()),
					zap.Int32("max", app.ConcurrencyLimiter.GetMaxConcurrency()),
				)
				utils.ServerBusy(w, "并发数超出限制")
				return
			}
			// 释放并发许可
			defer app.ConcurrencyLimiter.Release()
			next.ServeHTTP(w, r)
		})
	}
}
