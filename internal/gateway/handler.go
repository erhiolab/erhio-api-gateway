package gateway

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/middleware"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
)

// Handler 处理请求
func Handler(app *app.App) http.Handler {
	core := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		node, ok := r.Context().Value(utils.SelectedNodeKey).(*models.ServiceNode)
		if !ok {
			utils.BadGateway(w)
			return
		}
		Proxy(node.NodeURL, w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, ok := r.Context().Value(utils.RouteKey).(*models.Route)
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
func Build(route *models.Route, app *app.App) []middleware.Middleware {
	var mws []middleware.Middleware
	if route.RequireAuth {
		mws = append(
			mws,
			middleware.HeaderParser(),
			middleware.TimeWindow(),
			middleware.UniqueNonce(app),
			middleware.Authenticator(app),
			middleware.Authorizer(),
		)
	}
	if route.RequireLimit {
		mws = append(mws, middleware.RateLimit(app))
	}
	if route.IpLimit {
		mws = append(mws, middleware.IPLimit())
	}
	if route.CountryLimit {
		mws = append(mws, middleware.CountryLimit())
	}
	return mws
}
