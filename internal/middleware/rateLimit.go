package middleware

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// RateLimit 限流器插件
func RateLimit(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			route, ok := ctx.Value(utils.RouteKey).(*models.Route)
			if !ok {
				logger.WithRequestLogCtx(ctx).Error("限流器插件: 路由不存在",
					zap.Int64("route_id", route.ID),
					zap.String("path", route.Path),
					zap.String("method", route.Method),
				)
				utils.NotFound(w)
				return
			}
			service, ok := ctx.Value(utils.ServiceKey).(*models.Service)
			if !ok || len(service.Nodes) == 0 {
				logger.WithRequestLogCtx(ctx).Error("限流器插件: 服务没有活动节点, 无法限流",
					zap.Int64("service_id", service.ID),
					zap.String("service_name", service.Name),
				)
				utils.BadGateway(w)
				return
			}
			// 选择限流键
			apiKeyInfo := models.APIKeyInfo{}
			if apiKeyInfoPtr, ok := ctx.Value(utils.ApiKeyInfoKey).(*models.APIKeyInfo); ok && apiKeyInfoPtr != nil {
				apiKeyInfo = *apiKeyInfoPtr
			}
			ip, ok := ctx.Value(utils.ClientIPKey).(*models.IPLocation)
			if !ok || ip == nil {
				logger.WithRequestLogCtx(ctx).Error("限流器插件: 客户端IP不存在",
					zap.String("ip", ip.IP),
				)
				utils.BadRequest(w, "ip")
				return
			}
			cfg := config.Get()
			qpmKey := cfg.Redis.ProjectPrefix + ":limit:qpm:" + service.Name + ":" + route.Path
			qpsKey := cfg.Redis.ProjectPrefix + ":limit:qps:" + service.Name + ":" + route.Path
			var qpmLimit int64
			var qpsLimit int64
			if apiKeyInfo.SecretID != "" {
				qpmKey += ":key:" + apiKeyInfo.SecretID
				qpsKey += ":key:" + apiKeyInfo.SecretID
				qpmLimit = apiKeyInfo.QPM
				qpsLimit = apiKeyInfo.QPS
			} else {
				qpmKey += ":ip:" + ip.IP
				qpsKey += ":ip:" + ip.IP
				route := models.Route{}
				if routePtr, ok := ctx.Value(utils.RouteKey).(*models.Route); ok {
					route = *routePtr
				}
				qpmLimit = route.QPM
				qpsLimit = route.QPS
				if qpmLimit <= 0 {
					qpmLimit = cfg.DatabaseConfig.Auth.QpmLimit
				}
				if qpsLimit <= 0 {
					qpsLimit = cfg.DatabaseConfig.Auth.QpsLimit
				}
			}

			// 检查限流
			if qpmLimit > 0 {
				qpm, err := app.Redis.IncrAndExpire(qpmKey, time.Minute, false)
				if err != nil {
					logger.WithRequestLogCtx(ctx).Error("限流器插件: 增加QPM计数器失败",
						zap.String("ip", ip.IP),
						zap.Error(err),
					)
					utils.InternalServerError(w)
					return
				}
				if qpm > qpmLimit {
					logger.WithRequestLogCtx(ctx).Error("限流器插件: QPM限制超出",
						zap.String("ip", ip.IP),
						zap.Int64("qpm", qpm),
						zap.Int64("limit", qpmLimit),
					)
					utils.TooManyRequests(w)
					return
				}
			}
			if qpsLimit > 0 {
				qps, err := app.Redis.IncrAndExpire(qpsKey, time.Second, false)
				if err != nil {
					logger.WithRequestLogCtx(ctx).Error("限流器插件: 增加QPS计数器失败",
						zap.String("ip", ip.IP),
						zap.Error(err),
					)
					utils.InternalServerError(w)
					return
				}
				if qps > qpsLimit {
					logger.WithRequestLogCtx(ctx).Error("限流器插件: QPS限制超出",
						zap.String("ip", ip.IP),
						zap.Int64("qps", qps),
						zap.Int64("limit", qpsLimit),
					)
					utils.TooManyRequests(w)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
