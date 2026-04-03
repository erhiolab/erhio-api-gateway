package middleware

import (
	"context"
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/utils"
	"net/http"
)

// Router 路由插件
func Router(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 匹配 Service
			service, newPath, err := app.MatchService(r.URL.Path)
			if err != nil || service == nil || len(service.Nodes) == 0 {
				utils.NotFound(w)
				return
			}
			// 匹配 Route
			route, err := app.MatchRoute(service.ID, newPath, r.Method)
			if err != nil || route == nil || !route.Enabled {
				utils.NotFound(w)
				return
			}
			ctx := context.WithValue(r.Context(), utils.ServiceKey, service)
			ctx = context.WithValue(ctx, utils.RouteKey, route)
			ctx = context.WithValue(ctx, utils.UpstreamPathKey, newPath)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
