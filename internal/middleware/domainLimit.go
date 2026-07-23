package middleware

import (
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"strings"
)

// DomainLimit 域名限制器插件
func DomainLimit() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			// 获取请求域名
			host := r.Host
			if host == "" {
				host = r.URL.Host
			}
			// 移除端口号
			if idx := strings.LastIndex(host, ":"); idx != -1 {
				host = host[:idx]
			}
			if host == "" {
				logger.WithRequestLogCtx(ctx, r).Warn("域名限制器插件: 请求域名为空")
				utils.BadRequest(w, "域名为空")
				return
			}
			// 自定义域名限制
			if val := ctx.Value(utils.ApiKeyInfoKey); val != nil {
				if apiKeyInfo, ok := val.(*models.APIKeyInfo); ok {
					switch apiKeyInfo.DomainFilterType {
					case 1:
						if !isDomainInList(apiKeyInfo.DomainList, host) {
							logger.WithRequestLogCtx(ctx, r).Warn("域名限制器插件: 请求域名不在API密钥白名单中, 被拒绝访问")
							utils.Forbidden(w, "域名不在白名单中")
							return
						}
					case 2:
						if isDomainInList(apiKeyInfo.DomainList, host) {
							logger.WithRequestLogCtx(ctx, r).Warn("域名限制器插件: 请求域名在API密钥黑名单中, 被拒绝访问")
							utils.Forbidden(w, "域名被列入黑名单")
							return
						}
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isDomainInList 检查域名是否在列表中
func isDomainInList(list []string, host string) bool {
	for _, domain := range list {
		// 精确匹配
		if host == domain {
			return true
		}
		// 子域名匹配 (例如: example.com 匹配 www.example.com)
		if strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}
