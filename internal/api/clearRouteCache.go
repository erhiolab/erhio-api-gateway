package api

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/utils"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

// ClearRouteCache 清除路由缓存
func ClearRouteCache(app *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		routeIDStr := r.PathValue("route_id")
		var err error
		if routeIDStr == "" {
			err = app.ClearAllRouteCache()
		} else {
			routeID, parseErr := strconv.ParseInt(routeIDStr, 10, 64)
			if parseErr != nil {
				utils.BadRequest(w, "invalid route_id")
				return
			}
			err = app.ClearRouteCache(routeID)
		}
		if err != nil {
			logger.WithRequestLogCtx(ctx).Error("清除路由缓存失败",
				zap.String("route_id", routeIDStr),
				zap.Error(err),
			)
			utils.InternalServerError(w)
			return
		}
		utils.Success(w, "success")
	}
}
