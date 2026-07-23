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
			ctx := r.Context()
			authRequirement, ok := ctx.Value(utils.AuthRequirementKey).(*models.AuthRequirement)
			if !ok {
				logger.WithRequestLogCtx(ctx, r).Error("身份验证插件: 上下文中缺少认证要求")
				utils.InternalServerError(w, "上下文中缺少认证要求")
				return
			}
			if authRequirement.SecretID == "" {
				logger.WithRequestLogCtx(ctx, r).Warn("身份验证插件: Authorization header中缺少SecretID")
				utils.BadRequest(w, "授权为空")
				return
			}
			if authRequirement.Timestamp == "" {
				logger.WithRequestLogCtx(ctx, r).Warn("身份验证插件: X-Timestamp header中缺少时间戳")
				utils.BadRequest(w, "时间戳为空")
				return
			}
			if authRequirement.Nonce == "" {
				logger.WithRequestLogCtx(ctx, r).Warn("身份验证插件: X-Nonce header中缺少Nonce")
				utils.BadRequest(w, "Nonce为空")
				return
			}
			if authRequirement.Signature == "" {
				logger.WithRequestLogCtx(ctx, r).Warn("身份验证插件: X-Signature header中缺少签名")
				utils.BadRequest(w, "签名为空")
				return
			}
			apiKeyInfo, ok, err := app.GetApiKeyInfo(authRequirement.SecretID)
			if !ok {
				logger.WithRequestLogCtx(ctx, r).Error("身份验证插件: 上下文中缺少API密钥ID对应的API密钥信息",
					zap.String("SecretID", authRequirement.SecretID),
				)
				utils.Unauthorized(w, "API密钥无效")
				return
			}
			if err != nil {
				logger.WithRequestLogCtx(ctx, r).Error("身份验证插件: 查询API密钥信息异常",
					zap.String("SecretID", authRequirement.SecretID),
					zap.Error(err),
				)
				utils.InternalServerError(w, "查询API密钥信息异常")
				return
			}
			// 未启用API密钥, API密钥被封禁
			if !apiKeyInfo.Enabled || apiKeyInfo.Banned != 0 {
				logger.WithRequestLogCtx(ctx, r).Warn("身份验证插件: API密钥未启用或已被封禁",
					zap.String("SecretID", authRequirement.SecretID),
				)
				utils.Forbidden(w, "API密钥无效")
				return
			}
			// API密钥有过期时间, 且已过期
			if apiKeyInfo.ExpiresAt != nil && time.Now().After(*apiKeyInfo.ExpiresAt) {
				logger.WithRequestLogCtx(ctx, r).Warn("身份验证插件: API密钥已过期",
					zap.String("SecretID", authRequirement.SecretID),
				)
				utils.Forbidden(w, "API密钥无效")
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
				logger.WithRequestLogCtx(ctx, r).Warn("身份验证插件: 签名校验失败",
					zap.String("SecretID", authRequirement.SecretID),
					zap.String("Timestamp", authRequirement.Timestamp),
					zap.String("Nonce", authRequirement.Nonce),
					zap.String("Signature", authRequirement.Signature),
				)
				utils.Unauthorized(w, "签名校验失败")
				return
			}
			ctx = context.WithValue(ctx, utils.ApiKeyInfoKey, apiKeyInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
