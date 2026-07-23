package middleware

import (
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
)

// IPLimit IP限制器插件
func IPLimit() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			clientIP, ok := ctx.Value(utils.ClientIPKey).(*models.IPLocation)
			if !ok {
				logger.WithRequestLogCtx(ctx, r).Warn("IP限制器插件: 客户端IP信息不存在")
				utils.BadRequest(w, "客户端IP为空")
				return
			}
			// 自定义IP限制
			if val := ctx.Value(utils.ApiKeyInfoKey); val != nil {
				if apiKeyInfo, ok := val.(*models.APIKeyInfo); ok {
					switch apiKeyInfo.IPFilterType {
					case 1:
						if !utils.Contains(apiKeyInfo.IPList, clientIP.IP) {
							logger.WithRequestLogCtx(ctx, r).Warn("IP限制器插件: 客户端IP不在API Key白名单中, 被拒绝访问")
							utils.Forbidden(w, "IP不在白名单中")
							return
						}
					case 2:
						if utils.Contains(apiKeyInfo.IPList, clientIP.IP) {
							logger.WithRequestLogCtx(ctx, r).Warn("IP限制器插件: 客户端IP在API Key黑名单中, 被拒绝访问")
							utils.Forbidden(w, "IP被列入黑名单")
							return
						}
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
