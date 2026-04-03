package middleware

import (
	"context"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/utils"
	"net/http"
)

// Router 路由插件
func Router() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 匹配路由
			route := matchRoute(r.URL.Path, r.Method)
			if route == nil || !route.Enabled {
				utils.NotFound(w)
				return
			}
			// 获取服务
			service := getService(route.ServiceID)
			if service == nil || len(service.Nodes) == 0 {
				utils.BadGateway(w)
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), utils.RouteKey, route))
			r = r.WithContext(context.WithValue(r.Context(), utils.ServiceKey, service))
			next.ServeHTTP(w, r)
		})
	}
}

// matchRoute 匹配路由
func matchRoute(path string, method string) *config.Route {
	cfg := config.Get().Routes
	for i := range cfg {
		r := &cfg[i]
		if r.Path == path && r.Method == method {
			return r
		}
	}
	return nil
}

// getService 获取服务
func getService(id int64) *config.Service {
	cfg := config.Get().Services
	for i := range cfg {
		s := &cfg[i]
		if s.ID == id {
			return s
		}
	}
	return nil
}
