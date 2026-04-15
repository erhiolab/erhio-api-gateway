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
		// 获取网关信息
		services, err := app.DB.GetAllServices()
		if err != nil {
			logger.WithRequestLogCtx(ctx).Error("获取服务列表失败", zap.Error(err))
			utils.InternalServerError(w)
			return
		}
		// 计算所有服务节点数总和
		nodeCount := 0
		for _, service := range services {
			nodeCount += len(service.Nodes)
		}
		// 计算路由数
		routes, err := app.DB.GetAllRoutes()
		if err != nil {
			logger.WithRequestLogCtx(ctx).Error("获取路由列表失败", zap.Error(err))
			utils.InternalServerError(w)
			return
		}
		routeCount := len(routes)
		// 构建响应
		response := map[string]interface{}{
			"services": len(services),
			"nodes":    nodeCount,
			"routes":   routeCount,
		}
		utils.Success(w, response)
	}
}
