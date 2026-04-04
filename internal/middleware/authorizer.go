package middleware

import (
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
)

// Authorizer 鉴权插件
func Authorizer() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从上下文获取API密钥信息
			apiKeyInfo, ok := r.Context().Value(utils.ApiKeyInfoKey).(*models.APIKeyInfo)
			if !ok {
				utils.Unauthorized(w, "Invalid SecretID")
				return
			}
			route, ok := r.Context().Value(utils.RouteKey).(*models.Route)
			if !ok {
				utils.NotFound(w)
				return
			}
			if !utils.Contains(apiKeyInfo.RouteIDs, route.ID) {
				utils.Forbidden(w, "No access to this route")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
