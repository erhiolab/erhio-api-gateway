package middleware

import (
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
)

// Authorizer 鉴权插件
func Authorizer() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			apiKeyInfo, ok := ctx.Value(utils.ApiKeyInfoKey).(*models.APIKeyInfo)
			if !ok {
				logger.WithRequestLogCtx(ctx).Error("鉴权插件: 上下文中缺少API密钥信息")
				utils.Unauthorized(w, "API密钥无效")
				return
			}
			route, ok := ctx.Value(utils.RouteKey).(*models.Route)
			if !ok {
				logger.WithRequestLogCtx(ctx).Error("鉴权插件: 上下文中缺少路由信息")
				utils.NotFound(w, "路由不存在")
				return
			}
			if !utils.Contains(apiKeyInfo.RouteIDs, route.ID) {
				logger.WithRequestLogCtx(ctx).Warn("鉴权插件: API密钥未授权访问该路由")
				utils.Forbidden(w, "API密钥无效")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
