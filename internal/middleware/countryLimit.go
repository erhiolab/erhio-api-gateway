package middleware

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
)

// CountryLimit 国家限制器插件
func CountryLimit() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			clientIP, ok := ctx.Value(utils.ClientIPKey).(*models.IPLocation)
			if !ok {
				logger.WithRequestLogCtx(ctx).Error("国家限制器插件: 客户端IP信息不存在")
				utils.BadRequest(w, "ip")
				return
			}
			// 全局国家黑名单
			cfg := config.Get()
			if len(cfg.DatabaseConfig.Auth.CountryBlackList) > 0 && isCountryInList(cfg.DatabaseConfig.Auth.CountryBlackList, clientIP) {
				logger.WithRequestLogCtx(ctx).Error("国家限制器插件: 客户端IP在全局国家黑名单中, 被拒绝访问")
				utils.Forbidden(w, "Country is globally blacklisted")
				return
			}
			// 自定义国家限制
			if val := ctx.Value(utils.ApiKeyInfoKey); val != nil {
				if apiKeyInfo, ok := val.(*models.APIKeyInfo); ok {
					switch apiKeyInfo.CountryFilterType {
					case 1:
						if !isCountryInList(apiKeyInfo.CountryList, clientIP) {
							logger.WithRequestLogCtx(ctx).Error("国家限制器插件: 客户端IP不在API密钥白名单中, 被拒绝访问")
							utils.Forbidden(w, "Country not allowed by API Key whitelist")
							return
						}
					case 2:
						if isCountryInList(apiKeyInfo.CountryList, clientIP) {
							logger.WithRequestLogCtx(ctx).Error("国家限制器插件: 客户端IP在API密钥黑名单中, 被拒绝访问")
							utils.Forbidden(w, "Country blocked by API Key blacklist")
							return
						}
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isCountryInList 检查国家是否在列表中
func isCountryInList(list []string, loc *models.IPLocation) bool {
	return utils.Contains(list, loc.CountryShort) || utils.Contains(list, loc.CountryLong)
}
