package middleware

import (
	"context"
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Authenticator 身份验证插件
func Authenticator(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从上下文获取认证要求
			authRequirement := r.Context().Value(utils.AuthRequirementKey).(models.AuthRequirement)
			if authRequirement.SecretID == "" {
				utils.BadRequest(w, "Authorization")
				return
			}
			if authRequirement.Timestamp == "" {
				utils.BadRequest(w, "X-Timestamp")
				return
			}
			if authRequirement.Nonce == "" {
				utils.BadRequest(w, "X-Nonce")
				return
			}
			if authRequirement.Signature == "" {
				utils.BadRequest(w, "X-Signature")
				return
			}
			// 查询API密钥信息
			apiKeyInfo, ok, err := app.GetApiKeyInfo(authRequirement.SecretID)
			if !ok {
				utils.Unauthorized(w, "Invalid SecretID")
				return
			}
			if err != nil {
				logger.Log.Error("查询API密钥信息异常", zap.Error(err))
				utils.InternalServerError(w)
				return
			}
			// 未启用API密钥, API密钥被封禁
			if !apiKeyInfo.Enabled || apiKeyInfo.Banned != 0 {
				utils.Forbidden(w, "Key disabled or banned")
				return
			}
			// API密钥有过期时间, 且已过期
			if apiKeyInfo.ExpiresAt != nil && time.Now().After(*apiKeyInfo.ExpiresAt) {
				utils.Forbidden(w, "Key expired")
				return
			}
			// 校验签名
			if !utils.Verify(
				apiKeyInfo.SecretKey,
				r,
				authRequirement.Timestamp,
				authRequirement.Nonce,
				authRequirement.Signature,
			) {
				utils.Unauthorized(w, "Signature mismatch")
				return
			}
			ctx := context.WithValue(r.Context(), utils.ApiKeyInfoKey, apiKeyInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
