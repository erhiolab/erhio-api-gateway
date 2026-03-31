package middleware

import (
	"context"
	"elake-api-gateway/internal/utils"
	"net/http"

	"github.com/google/uuid"
)

// RequestID 中间件, 为每个请求添加一个唯一的请求ID到上下文键
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := uuid.New().String()
			ctx := context.WithValue(r.Context(), utils.RequestIDKey, requestID)
			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
