package middleware

import (
	"elake-api-gateway/internal/healthManager"
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
					utils.InternalServerError(w)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
