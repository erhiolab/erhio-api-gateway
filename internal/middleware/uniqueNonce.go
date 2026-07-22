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

// UniqueNonce 唯一性Nonce插件
func UniqueNonce(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			cfg := config.Get()
			// 从上下文获取认证要求
			authRequirement, ok := ctx.Value(utils.AuthRequirementKey).(*models.AuthRequirement)
			if !ok || authRequirement.SecretID == "" {
				logger.WithRequestLogCtx(ctx).Warn("唯一性Nonce插件: Authorization header中缺少SecretID")
				utils.BadRequest(w, "授权为空")
				return
			}
			if authRequirement.Nonce == "" {
				logger.WithRequestLogCtx(ctx).Warn("唯一性Nonce插件: X-Nonce header中缺少Nonce值")
				utils.BadRequest(w, "Nonce为空")
				return
			}
			nonceKey := cfg.Redis.ProjectPrefix + ":limit:nonce:" + authRequirement.SecretID + ":" + authRequirement.Nonce
			count, err := app.Redis.IncrAndExpire(nonceKey, time.Duration(cfg.DatabaseConfig.Auth.NonceWindowHour)*time.Hour, false)
			if err != nil {
				logger.WithRequestLogCtx(ctx).Warn("唯一性Nonce插件: Nonce窗口错误",
					zap.String("nonce", authRequirement.Nonce),
					zap.Error(err),
				)
				utils.InternalServerError(w, "Nonce窗口异常")
				return
			}
			if count > cfg.DatabaseConfig.Auth.NonceWindow {
				logger.WithRequestLogCtx(ctx).Warn("唯一性Nonce插件: Nonce窗口超出范围",
					zap.String("nonce", authRequirement.Nonce),
					zap.Int64("count", count),
					zap.Int64("window", cfg.DatabaseConfig.Auth.NonceWindow),
				)
				utils.Forbidden(w, "Nonce窗口超出范围")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
