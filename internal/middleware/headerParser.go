package middleware

import (
	"context"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"strings"
)

// HeaderParser 头解析插件
func HeaderParser() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				logger.WithRequestLogCtx(ctx).Warn("头解析插件: Authorization header中缺少SecretID")
				utils.BadRequest(w, "授权为空")
				return
			}
			secretID := strings.TrimPrefix(authHeader, "Bearer ")
			timestamp := r.Header.Get("X-Timestamp")
			if timestamp == "" {
				logger.WithRequestLogCtx(ctx).Warn("头解析插件: X-Timestamp header中缺少时间戳")
				utils.BadRequest(w, "时间戳为空")
				return
			}
			nonce := r.Header.Get("X-Nonce")
			if nonce == "" {
				logger.WithRequestLogCtx(ctx).Warn("头解析插件: X-Nonce header中缺少Nonce值")
				utils.BadRequest(w, "Nonce为空")
				return
			}
			signature := r.Header.Get("X-Signature")
			if signature == "" {
				logger.WithRequestLogCtx(ctx).Warn("头解析插件: X-Signature header中缺少签名")
				utils.BadRequest(w, "签名为空")
				return
			}
			authRequirement := &models.AuthRequirement{
				SecretID:  secretID,
				Timestamp: timestamp,
				Nonce:     nonce,
				Signature: signature,
			}
			ctx = context.WithValue(ctx, utils.AuthRequirementKey, authRequirement)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
