package middleware

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"time"
)

// UniqueNonce 唯一性Nonce插件
func UniqueNonce(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := config.Get()
			// 从上下文获取认证要求
			authRequirement := r.Context().Value(utils.AuthRequirementKey).(models.AuthRequirement)
			if authRequirement.SecretID == "" {
				utils.BadRequest(w, "Authorization")
				return
			}
			if authRequirement.Nonce == "" {
				utils.BadRequest(w, "X-Nonce")
				return
			}
			nonceKey := cfg.Redis.ProjectPrefix + ":limit:nonce:" + authRequirement.SecretID + ":" + authRequirement.Nonce
			count, err := app.Redis.IncrAndExpire(nonceKey, time.Duration(cfg.DatabaseConfig.Auth.NonceWindowHour)*time.Hour, false)
			if err != nil {
				utils.InternalServerError(w)
				return
			}
			if count > cfg.DatabaseConfig.Auth.NonceWindow {
				utils.Forbidden(w, "Nonce window exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
