package middleware

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// TimeWindow 时间窗口插件
func TimeWindow() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			cfg := config.Get()
			// 从上下文获取认证要求
			authRequirement, ok := ctx.Value(utils.AuthRequirementKey).(*models.AuthRequirement)
			if !ok || authRequirement.Timestamp == "" {
				logger.WithRequestLogCtx(ctx).Warn("时间窗口插件: X-Timestamp header中缺少时间戳")
				utils.BadRequest(w, "empty X-Timestamp")
				return
			}
			// 校验时间戳
			ts, err := strconv.ParseInt(authRequirement.Timestamp, 10, 64)
			if err != nil {
				logger.WithRequestLogCtx(ctx).Warn("时间窗口插件: 时间戳格式错误",
					zap.String("timestamp", authRequirement.Timestamp),
					zap.Error(err),
				)
				utils.Unauthorized(w)
				return
			}
			if len(authRequirement.Timestamp) == 13 {
				ts /= 1000
			}
			// 校验时间窗口
			if utils.Abs(time.Now().Unix()-ts) > cfg.DatabaseConfig.Auth.TimestampWindow {
				logger.WithRequestLogCtx(ctx).Warn("时间窗口插件: 请求插件: 时间戳超出窗口范围",
					zap.Int64("timestamp", ts),
					zap.Int64("window", cfg.DatabaseConfig.Auth.TimestampWindow),
				)
				utils.Forbidden(w, "Request expired")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
