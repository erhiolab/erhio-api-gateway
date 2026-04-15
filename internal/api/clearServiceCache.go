package api

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/utils"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

// ClearServiceCache 清除服务缓存
func ClearServiceCache(app *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		serviceIDStr := r.PathValue("service_id")
		var err error
		if serviceIDStr == "" {
			err = app.ClearAllServiceCache()
		} else {
			serviceID, parseErr := strconv.ParseInt(serviceIDStr, 10, 64)
			if parseErr != nil {
				utils.BadRequest(w, "invalid service_id")
				return
			}
			err = app.ClearServiceCache(serviceID)
		}
		if err != nil {
			logger.WithRequestLogCtx(ctx).Error("清除服务缓存失败",
				zap.String("service_id", serviceIDStr),
				zap.Error(err),
			)
			utils.InternalServerError(w)
			return
		}
		utils.Success(w, "success")
	}
}
