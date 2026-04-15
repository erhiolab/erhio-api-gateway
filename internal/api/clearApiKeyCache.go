package api

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/utils"
	"net/http"

	"go.uber.org/zap"
)

// ClearApiKeyCache 清除API密钥缓存
func ClearApiKeyCache(app *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		secretId := r.PathValue("secret_id")
		var err error
		if secretId == "" {
			err = app.ClearAllApiKeyInfoCache()
		} else {
			err = app.ClearApiKeyInfoCache(secretId)
		}
		if err != nil {
			logger.WithRequestLogCtx(ctx).Error("清除API密钥缓存失败",
				zap.String("secret_id", secretId),
				zap.Error(err),
			)
			utils.InternalServerError(w)
			return
		}
		utils.Success(w, "success")
	}
}
