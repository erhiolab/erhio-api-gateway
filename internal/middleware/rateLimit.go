package middleware

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/utils"
	"net/http"
	"time"
)

// RateLimit 限流中间件
func RateLimit(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从上下文获取路由信息
			route, ok := r.Context().Value(utils.RouteKey).(*config.Route)
			if !ok {
				utils.Error(w, http.StatusNotFound, 4040, "Not Found")
				return
			}
			service, ok := r.Context().Value(utils.ServiceKey).(*config.Service)
			if !ok || len(service.Nodes) == 0 {
				utils.Error(w, http.StatusBadGateway, 5020, "service unavailable")
				return
			}

			// 选择限流键
			secretID := ""
			ip, _ := r.Context().Value(utils.ClientIPKey).(string)
			qpsKey := "rate_limit:" + service.Name + ":" + route.Path
			qpmKey := "rate_limit:" + service.Name + ":" + route.Path
			var qpsLimit int64
			var qpmLimit int64
			if secretID != "" {
				qpsKey += ":key:qps:" + secretID
				qpmKey += ":key:qpm:" + secretID
				qpsLimit = 100
				qpmLimit = 100
			} else {
				cfg := config.Get().Gateway
				qpsKey += ":ip:qps:" + ip
				qpmKey += ":ip:qpm:" + ip
				qpsLimit = cfg.QpsLimit
				qpmLimit = cfg.QpmLimit
			}

			// 检查限流
			qps, err := app.Redis.IncrAndExpire(qpsKey, time.Second, false)
			if err != nil {
				utils.Error(w, http.StatusInternalServerError, 5000, "rate limit error")
				return
			}
			if qps > qpsLimit {
				utils.Error(w, http.StatusTooManyRequests, 4029, "rate limit exceeded")
				return
			}
			qpm, err := app.Redis.IncrAndExpire(qpmKey, time.Minute, true)
			if err != nil {
				utils.Error(w, http.StatusInternalServerError, 5000, "rate limit error")
				return
			}
			if qpm > qpmLimit {
				utils.Error(w, http.StatusTooManyRequests, 4029, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
