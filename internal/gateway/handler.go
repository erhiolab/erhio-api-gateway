package gateway

import (
	"elake-api-gateway/internal/config"
	"net/http"
)

// CoreHandler 处理请求
func CoreHandler(w http.ResponseWriter, r *http.Request) {
	route := matchRoute(r.URL.Path, r.Method)
	// 404 - 未匹配到路由
	if route == nil {
		http.NotFound(w, r)
		return
	}
	// 502 - 服务不可用
	service := getService(route.Service)
	if service == nil || len(service.Nodes) == 0 {
		http.Error(w, "service unavailable", http.StatusBadGateway)
		return
	}
	node := service.Nodes[0]
	Proxy(node, w, r)
}

// matchRoute 匹配路由
func matchRoute(path string, method string) *config.Route {
	cfg := config.Get()
	for _, r := range cfg.Routes {
		if r.Path == path && r.Method == method {
			return &r
		}
	}
	return nil
}

// getService 获取服务
func getService(name string) *config.Service {
	cfg := config.Get()
	for _, s := range cfg.Services {
		if s.Name == name {
			return &s
		}
	}
	return nil
}
