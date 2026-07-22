package middleware

import (
	"context"
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/utils"
	"net/http"

	"go.uber.org/zap"
)

// Router 路由插件
func Router(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			// 匹配 Service
			service, newPath, err := app.MatchService(r.URL.Path)
			if err != nil || service == nil || len(service.Nodes) == 0 {
				logger.WithRequestLogCtx(ctx).Warn("路由插件: 服务未找到",
					zap.String("path", r.URL.Path),
					zap.Error(err),
				)
				utils.NotFound(w, "服务不存在")
				return
			}
			// 匹配 Route
			route, err := app.MatchRoute(service.ID, newPath, r.Method)
			if err != nil || route == nil || !route.Enabled {
				logger.WithRequestLogCtx(ctx).Warn("路由插件: 路由未找到",
					zap.String("path", r.URL.Path),
					zap.Error(err),
				)
				utils.NotFound(w, "路由不存在")
				return
			}
			ctx = context.WithValue(ctx, utils.ServiceKey, service)
			ctx = context.WithValue(ctx, utils.RouteKey, route)
			ctx = context.WithValue(ctx, utils.UpstreamPathKey, newPath)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
