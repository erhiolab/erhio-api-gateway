package gateway

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/controller"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/utils"
	"net/http"
)

// APIHandler 处理网关自身的API请求
func APIHandler(app *app.App) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", controller.Ping(app))
	mux.HandleFunc("POST /publish_message", controller.PublishMessage(app))
	mux.HandleFunc("GET /ip/region", controller.IPRegion(app))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger.WithRequestLogCtx(ctx, r).Warn("处理请求: 路由不存在")
		utils.NotFound(w, "路由不存在")
	})
	return mux
}
