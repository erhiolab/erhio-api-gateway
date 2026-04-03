package middleware

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"time"
)

// RateLimit 限流器插件
func RateLimit(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route, ok := r.Context().Value(utils.RouteKey).(*models.Route)
			if !ok {
				utils.NotFound(w)
				return
			}
			service, ok := r.Context().Value(utils.ServiceKey).(*models.Service)
			if !ok || len(service.Nodes) == 0 {
				utils.BadGateway(w)
				return
			}
			// 选择限流键
			apiKeyInfo := models.APIKeyInfo{}
			if apiKeyInfoPtr, ok := r.Context().Value(utils.ApiKeyInfoKey).(*models.APIKeyInfo); ok && apiKeyInfoPtr != nil {
				apiKeyInfo = *apiKeyInfoPtr
			}
			ip, ok := r.Context().Value(utils.ClientIPKey).(*models.IPLocation)
			if !ok || ip == nil {
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
				if routePtr, ok := r.Context().Value(utils.RouteKey).(*models.Route); ok {
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
					utils.InternalServerError(w)
					return
				}
				if qpm > qpmLimit {
					utils.TooManyRequests(w)
					return
				}
			}
			if qpsLimit > 0 {
				qps, err := app.Redis.IncrAndExpire(qpsKey, time.Second, false)
				if err != nil {
					utils.InternalServerError(w)
					return
				}
				if qps > qpsLimit {
					utils.TooManyRequests(w)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
