package middleware

import (
	"elake-api-gateway/internal/service/healthManager"
	"elake-api-gateway/internal/utils"
	"net/http"
)

// HealthCheck 健康检查插件
func HealthCheck() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			deps := []string{"DB", "Redis", "IPDB"}
			for _, dep := range deps {
				if !healthManager.Global().IsHealthy(dep) {
					utils.InternalServerError(w, dep+"健康检查失败")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
