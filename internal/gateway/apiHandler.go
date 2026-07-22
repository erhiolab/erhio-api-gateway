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

	// 发布消息
	mux.HandleFunc("POST /publish_message", api.PublishMessage(app))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger.WithRequestLogCtx(ctx).Warn("处理请求: 路由不存在")
		utils.NotFound(w, "路由不存在")
	})
	return mux
}
