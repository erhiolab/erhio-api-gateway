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
		servicesCount, nodesCount, routesCount, err := app.DB.GetDashboardStats()
		if err != nil {
			logger.WithRequestLogCtx(ctx).Error("统计仪表盘信息错误", zap.Error(err))
			utils.InternalServerError(w, "统计仪表盘信息失败")
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
