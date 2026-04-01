package middleware

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/utils"
	"net/http"
	"time"
)

// RateLimit 限流器插件
func RateLimit(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从上下文获取路由信息
			route, ok := r.Context().Value(utils.RouteKey).(*config.Route)
			if !ok {
				utils.NotFound(w)
				return
			}
			service, ok := r.Context().Value(utils.ServiceKey).(*config.Service)
			if !ok || len(service.Nodes) == 0 {
				utils.BadGateway(w)
				return
			}

			// 选择限流键
			secretID := ""
			ip, ok := r.Context().Value(utils.ClientIPKey).(string)
			if !ok || ip == "" {
				utils.BadRequest(w, "ip")
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
				cfg := config.Get().Auth
				qpmKey += ":ip:" + ip
				qpsKey += ":ip:" + ip
				qpmLimit = cfg.QpmLimit
				qpsLimit = cfg.QpsLimit
			}

			// 检查限流
			qpm, err := app.Redis.IncrAndExpire(qpmKey, time.Minute, false)
			if err != nil {
				utils.InternalServerError(w)
				return
			}
			if qpm > qpmLimit {
				utils.TooManyRequests(w)
				return
			}
			qps, err := app.Redis.IncrAndExpire(qpsKey, time.Second, false)
			if err != nil {
				utils.InternalServerError(w)
				return
			}
			if qps > qpsLimit {
				utils.TooManyRequests(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
