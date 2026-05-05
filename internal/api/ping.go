package api

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/utils"
	"net/http"

	"go.uber.org/zap"
)

// Ping ping
func Ping(app *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		servicesCount, err := app.DB.GetServicesCount()
		if err != nil {
			logger.WithRequestLogCtx(ctx).Error("获取服务总数失败", zap.Error(err))
			utils.InternalServerError(w)
			return
		}
		nodesCount, err := app.DB.GetServiceNodesCount()
		if err != nil {
			logger.WithRequestLogCtx(ctx).Error("获取节点总数失败", zap.Error(err))
			utils.InternalServerError(w)
			return
		}
		routesCount, err := app.DB.GetRoutesCount()
		if err != nil {
			logger.WithRequestLogCtx(ctx).Error("获取路由总数失败", zap.Error(err))
			utils.InternalServerError(w)
			return
		}
		// 构建响应
		response := map[string]interface{}{
			"services": servicesCount,
			"nodes":    nodesCount,
			"routes":   routesCount,
		}
		utils.Success(w, response)
	}
}
