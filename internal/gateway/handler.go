package gateway

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/middleware"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
)

// Handler 处理请求
func Handler(app *app.App) http.Handler {
	core := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		selectedNode, ok := ctx.Value(utils.SelectedNodeKey).(*models.SelectedNode)
		if !ok || selectedNode == nil || selectedNode.Node == nil {
			logger.WithRequestLogCtx(ctx).Warn("处理请求: 未选择服务节点")
			utils.BadGateway(w)
			return
		}
		service, ok := ctx.Value(utils.ServiceKey).(*models.Service)
		if !ok || service == nil {
			logger.WithRequestLogCtx(ctx).Warn("处理请求: 未找到服务信息")
			utils.BadGateway(w)
			return
		}
		Proxy(service, selectedNode, w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		route, ok := ctx.Value(utils.RouteKey).(*models.Route)
		if !ok {
			logger.WithRequestLogCtx(ctx).Warn("处理请求: 路由不存在")
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
	if route.IpLimit {
		mws = append(mws, middleware.IPLimit())
	}
	if route.CountryLimit {
		mws = append(mws, middleware.CountryLimit())
	}
	return mws
}
