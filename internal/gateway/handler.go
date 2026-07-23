package gateway

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/middleware"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"fmt"
	"net/http"
	"sync"
)

// middlewareCache 中间件缓存
var middlewareCache = struct {
	sync.RWMutex
	m map[string][]middleware.Middleware
}{m: make(map[string][]middleware.Middleware)}

// Handler 处理请求
func Handler(app *app.App) http.Handler {
	core := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		selectedNode, ok := ctx.Value(utils.SelectedNodeKey).(*models.SelectedNode)
		if !ok || selectedNode == nil || selectedNode.Node == nil {
			logger.WithRequestLogCtx(ctx, r).Warn("处理请求: 未选择服务节点")
			utils.BadGateway(w, "未选择服务节点")
			return
		}
		service, ok := ctx.Value(utils.ServiceKey).(*models.Service)
		if !ok || service == nil {
			logger.WithRequestLogCtx(ctx, r).Warn("处理请求: 未找到服务信息")
			utils.BadGateway(w, "未找到服务信息")
			return
		}
		Proxy(service, selectedNode, w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		route, ok := ctx.Value(utils.RouteKey).(*models.Route)
		if !ok {
			logger.WithRequestLogCtx(ctx, r).Warn("处理请求: 路由不存在")
			utils.NotFound(w, "路由不存在")
			return
		}
		mws := getOrBuildMiddleware(route, app)
		handler := middleware.Chain(core, mws...)
		handler.ServeHTTP(w, r)
	})
}

// buildMiddlewareCacheKey 构建中间件缓存键
func buildMiddlewareCacheKey(route *models.Route) string {
	return fmt.Sprintf("%d:a%d:i%d:c%d:d%d", route.ID,
		boolToInt(route.RequireAuth),
		boolToInt(route.IpLimit),
		boolToInt(route.CountryLimit),
		boolToInt(route.DomainLimit))
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// getOrBuildMiddleware 获取或构建中间件链（带缓存）
func getOrBuildMiddleware(route *models.Route, app *app.App) []middleware.Middleware {
	cacheKey := buildMiddlewareCacheKey(route)
	// 先尝试读缓存
	middlewareCache.RLock()
	if mws, ok := middlewareCache.m[cacheKey]; ok {
		middlewareCache.RUnlock()
		return mws
	}
	middlewareCache.RUnlock()
	// 缓存未命中, 构建中间件链
	middlewareCache.Lock()
	defer middlewareCache.Unlock()
	// 双重检查, 避免并发构建
	if mws, ok := middlewareCache.m[cacheKey]; ok {
		return mws
	}
	mws := build(route, app)
	middlewareCache.m[cacheKey] = mws
	return mws
}

// build 构建中间件链
func build(route *models.Route, app *app.App) []middleware.Middleware {
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
	if route.DomainLimit {
		mws = append(mws, middleware.DomainLimit())
	}
	return mws
}
