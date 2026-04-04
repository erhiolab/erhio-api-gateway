package middleware

import (
	"context"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"strings"
)

// HeaderParser 头解析插件
func HeaderParser() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				utils.BadRequest(w, "Authorization")
				return
			}
			secretID := strings.TrimPrefix(authHeader, "Bearer ")
			nonce := r.Header.Get("X-Nonce")
			if nonce == "" {
				utils.BadRequest(w, "X-Nonce")
				return
			}
			timestamp := r.Header.Get("X-Timestamp")
			if timestamp == "" {
				utils.BadRequest(w, "X-Timestamp")
				return
			}
			signature := r.Header.Get("X-Signature")
			if signature == "" {
				utils.BadRequest(w, "X-Signature")
				return
			}
			authRequirement := models.AuthRequirement{
				SecretID:  secretID,
				Timestamp: timestamp,
				Nonce:     nonce,
				Signature: signature,
			}
			ctx := context.WithValue(r.Context(), utils.AuthRequirementKey, authRequirement)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
