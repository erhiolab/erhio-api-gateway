package middleware

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
)

// IPLimit IP限制器插件
func IPLimit() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP, ok := r.Context().Value(utils.ClientIPKey).(*models.IPLocation)
			if !ok {
				utils.BadRequest(w, "ip")
				return
			}
			// 全局IP黑名单
			cfg := config.Get()
			if len(cfg.DatabaseConfig.Auth.IPBlacklist) > 0 && utils.Contains(cfg.DatabaseConfig.Auth.IPBlacklist, clientIP.IP) {
				utils.Forbidden(w, "IP is globally blacklisted")
				return
			}
			// 自定义IP限制
			if val := r.Context().Value(utils.ApiKeyInfoKey); val != nil {
				if apiKeyInfo, ok := val.(*models.APIKeyInfo); ok {
					switch apiKeyInfo.IPFilterType {
					case 1:
						if !utils.Contains(apiKeyInfo.IPList, clientIP.IP) {
							utils.Forbidden(w, "IP not allowed by API Key whitelist")
							return
						}
					case 2:
						if utils.Contains(apiKeyInfo.IPList, clientIP.IP) {
							utils.Forbidden(w, "IP blocked by API Key blacklist")
							return
						}
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
