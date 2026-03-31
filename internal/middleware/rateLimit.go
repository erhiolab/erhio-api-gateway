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
			ip, ok := r.Context().Value(utils.ClientIPKey).(string)
			if !ok || ip == "" {
				utils.Error(w, http.StatusBadRequest, 4000, "client ip is empty")
				return
			}
			qpmKey := "rl:qpm:" + service.Name + ":" + route.Path
			qpsKey := "rl:qps:" + service.Name + ":" + route.Path
			var qpmLimit int64
			var qpsLimit int64
			// TODO 等待鉴权系统加入后, 从数据库获取限流配置
			if secretID != "" {
				qpmKey += ":key:" + secretID
				qpsKey += ":key:" + secretID
				qpmLimit = 100
				qpsLimit = 100
			} else {
				cfg := config.Get().Gateway
				qpmKey += ":ip:" + ip
				qpsKey += ":ip:" + ip
				qpmLimit = cfg.QpmLimit
				qpsLimit = cfg.QpsLimit
			}

			// 检查限流
			qpm, err := app.Redis.IncrAndExpire(qpmKey, time.Minute, false)
			if err != nil {
				utils.Error(w, http.StatusInternalServerError, 5000, "rate limit error")
				return
			}
			if qpm > qpmLimit {
				utils.Error(w, http.StatusTooManyRequests, 4029, "rate limit exceeded")
				return
			}
			qps, err := app.Redis.IncrAndExpire(qpsKey, time.Second, false)
			if err != nil {
				utils.Error(w, http.StatusInternalServerError, 5000, "rate limit error")
				return
			}
			if qps > qpsLimit {
				utils.Error(w, http.StatusTooManyRequests, 4029, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
