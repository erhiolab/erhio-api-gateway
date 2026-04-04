package middleware

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"strconv"
	"time"
)

// TimeWindow 时间窗口插件
func TimeWindow() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := config.Get()
			// 从上下文获取认证要求
			authRequirement := r.Context().Value(utils.AuthRequirementKey).(models.AuthRequirement)
			if authRequirement.Timestamp == "" {
				utils.BadRequest(w, "X-Timestamp")
				return
			}
			// 校验时间戳
			ts, err := strconv.ParseInt(authRequirement.Timestamp, 10, 64)
			if err != nil {
				utils.Unauthorized(w)
				return
			}
			if len(authRequirement.Timestamp) == 13 {
				ts /= 1000
			}
			// 校验时间窗口
			if utils.Abs(time.Now().Unix()-ts) > cfg.DatabaseConfig.Auth.TimestampWindow {
				utils.Forbidden(w, "Request expired")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
