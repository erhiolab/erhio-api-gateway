package gateway

import (
	"elake-api-gateway/internal/api"
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/utils"
	"net/http"
)

// APIHandler 处理网关自身的API请求
func APIHandler(app *app.App) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", api.Ping(app))

	// 清除API密钥缓存
	mux.HandleFunc("DELETE /clear_api_key_cache", api.ClearApiKeyCache(app))
	mux.HandleFunc("DELETE /clear_api_key_cache/{secret_id}", api.ClearApiKeyCache(app))
	// 清除服务缓存
	mux.HandleFunc("DELETE /clear_service_cache", api.ClearServiceCache(app))
	mux.HandleFunc("DELETE /clear_service_cache/{service_id}", api.ClearServiceCache(app))
	// 清除路由缓存
	mux.HandleFunc("DELETE /clear_route_cache", api.ClearRouteCache(app))
	mux.HandleFunc("DELETE /clear_route_cache/{route_id}", api.ClearRouteCache(app))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger.WithRequestLogCtx(ctx).Warn("处理请求: 路由不存在")
		utils.NotFound(w)
	})
	return mux
}
