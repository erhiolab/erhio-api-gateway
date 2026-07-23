package middleware

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// GlobalBlacklist 全局黑名单插件
func GlobalBlacklist(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			// 获取客户端IP信息
			clientIP, ok := ctx.Value(utils.ClientIPKey).(*models.IPLocation)
			if !ok {
				logger.WithRequestLogCtx(ctx, r).Warn("全局黑名单插件: 客户端IP信息不存在")
				utils.BadRequest(w, "客户端IP为空")
				return
			}
			// 检查全局IP黑名单
			ipBlacklist, err := app.GetBlacklistByType("ip")
			if err != nil {
				logger.WithRequestLogCtx(ctx, r).Error("全局黑名单插件: 获取全局IP黑名单失败", zap.Error(err))
			} else if len(ipBlacklist) > 0 {
				ipValues := make([]string, 0, len(ipBlacklist))
				for _, item := range ipBlacklist {
					ipValues = append(ipValues, item.Value)
				}
				if utils.Contains(ipValues, clientIP.IP) {
					logger.WithRequestLogCtx(ctx, r).Warn("全局黑名单插件: 客户端IP在全局IP黑名单中, 被拒绝访问")
					utils.Forbidden(w, "IP被列入黑名单")
					return
				}
			}
			// 检查全局国家黑名单
			countryBlacklist, err := app.GetBlacklistByType("country")
			if err != nil {
				logger.WithRequestLogCtx(ctx, r).Error("全局黑名单插件: 获取全局国家黑名单失败", zap.Error(err))
			} else if len(countryBlacklist) > 0 {
				countryValues := make([]string, 0, len(countryBlacklist))
				for _, item := range countryBlacklist {
					countryValues = append(countryValues, item.Value)
				}
				if isCountryInList(countryValues, clientIP) {
					logger.WithRequestLogCtx(ctx, r).Warn("全局黑名单插件: 客户端IP在全局国家黑名单中, 被拒绝访问")
					utils.Forbidden(w, "国家被列入黑名单")
					return
				}
			}
			// 获取请求域名
			host := r.Host
			if host == "" {
				host = r.URL.Host
			}
			// 移除端口号
			if idx := strings.LastIndex(host, ":"); idx != -1 {
				host = host[:idx]
			}
			if host != "" {
				// 检查全局域名黑名单
				domainBlacklist, err := app.GetBlacklistByType("domain")
				if err != nil {
					logger.WithRequestLogCtx(ctx, r).Error("全局黑名单插件: 获取全局域名黑名单失败", zap.Error(err))
				} else if len(domainBlacklist) > 0 {
					domainValues := make([]string, 0, len(domainBlacklist))
					for _, item := range domainBlacklist {
						domainValues = append(domainValues, item.Value)
					}
					if isDomainInList(domainValues, host) {
						logger.WithRequestLogCtx(ctx, r).Warn("全局黑名单插件: 请求域名在全局域名黑名单中, 被拒绝访问",
							zap.String("host", host))
						utils.Forbidden(w, "域名被列入黑名单")
						return
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
