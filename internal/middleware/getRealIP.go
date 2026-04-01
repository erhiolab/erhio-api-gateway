package middleware

import (
	"context"
	"elake-api-gateway/internal/utils"
	"net"
	"net/http"
	"strings"
)

// GetRealIP IP解析插件
func GetRealIP() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := GetClientIP(r)
			ctx := context.WithValue(r.Context(), utils.ClientIPKey, ip)
			if ip == "" {
				utils.BadRequest(w, "ip")
				return
			}
			r.Header.Set("X-Real-IP", ip)
			// 追加到 X-Forwarded-For
			xff := r.Header.Get("X-Forwarded-For")
			if xff == "" {
				r.Header.Set("X-Forwarded-For", ip)
			} else {
				r.Header.Set("X-Forwarded-For", xff+", "+ip)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClientIP 获取客户端 IP 地址
func GetClientIP(r *http.Request) string {
	// X-Forwarded-For
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		for _, part := range parts {
			ip := strings.TrimSpace(part)
			if parsed := net.ParseIP(ip); parsed != nil {
				return parsed.String()
			}
		}
	}

	// X-Real-IP
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		if parsed := net.ParseIP(ip); parsed != nil {
			return parsed.String()
		}
	}
	// RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		if parsed := net.ParseIP(host); parsed != nil {
			return parsed.String()
		}
	}
	return ""
}
