package middleware

import (
	"context"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/utils"
	"net/http"
	"strings"
)

// Router 路由插件
func Router() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 匹配 Service
			service, newPath := matchService(r.URL.Path)
			if service == nil || len(service.Nodes) == 0 {
				utils.NotFound(w)
				return
			}
			// 匹配 Route
			route := matchRoute(service.ID, newPath, r.Method)
			if route == nil || !route.Enabled {
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

// matchRoute 匹配路由
func matchRoute(serviceID int64, path string, method string) *config.Route {
	cfg := config.Get().Routes
	for i := range cfg {
		r := &cfg[i]
		if r.ServiceID == serviceID &&
			r.Path == path &&
			r.Method == method {
			return r
		}
	}
	return nil
}

// matchService 匹配服务
func matchService(path string) (*config.Service, string) {
	services := config.Get().Services
	for i := range services {
		s := &services[i]
		if path == s.BasePath || strings.HasPrefix(path, s.BasePath+"/") {
			newPath := strings.TrimPrefix(path, s.BasePath)
			if newPath == "" {
				newPath = "/"
			}
			return s, newPath
		}
	}
	return nil, ""
}
