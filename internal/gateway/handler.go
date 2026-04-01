package gateway

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/middleware"
	"elake-api-gateway/internal/utils"
	"net/http"
)

// Handler 处理请求
func Handler(app *app.App) http.Handler {
	core := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		service, ok := r.Context().Value(utils.ServiceKey).(*config.Service)
		if !ok {
			utils.BadGateway(w)
			return
		}
		node := service.Nodes[0]
		Proxy(node.NodeURL, w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, ok := r.Context().Value(utils.RouteKey).(*config.Route)
		if !ok {
			utils.NotFound(w)
			return
		}
		mws := Build(route, app)
		handler := middleware.Chain(core, mws...)
		handler.ServeHTTP(w, r)
	})
}

// Build 构建中间件链
func Build(route *config.Route, app *app.App) []middleware.Middleware {
	var mws []middleware.Middleware
	if route.RequireAuth {
		mws = append(mws, middleware.Auth())
	}
	if route.RequireLimit {
		mws = append(mws, middleware.RateLimit(app))
	}
	return mws
}
