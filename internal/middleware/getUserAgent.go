package middleware

import (
	"context"
	"elake-api-gateway/internal/utils"
	"net/http"
	"strings"
)

// GetUserAgent User-Agent解析插件
func GetUserAgent() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ua := r.Header.Get("User-Agent")
			if ua == "" {
				ua = "unknown"
			}
			device := ParseUserAgent(ua)
			ctx := context.WithValue(r.Context(), utils.UserAgentKey, utils.UserAgent{UserAgent: ua, Device: device})
			r.Header.Set("User-Agent", ua)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ParseUserAgent 解析客户端 User-Agent
func ParseUserAgent(ua string) string {
	if ua == "" {
		return "Unknown"
	}
	ua = strings.ToLower(ua)
	switch {
	// API 工具
	case strings.Contains(ua, "apifox"):
		return "Apifox"
	case strings.Contains(ua, "apidog"):
		return "Apidog"
	case strings.Contains(ua, "postman"):
		return "Postman"
	case strings.Contains(ua, "insomnia"):
		return "Insomnia"
	case strings.Contains(ua, "curl"):
		return "Curl"
	// Apple
	case strings.Contains(ua, "iphone"):
		return "iPhone"
	case strings.Contains(ua, "ipad"):
		return "iPad"
	case strings.Contains(ua, "macintosh"):
		return "Mac"
	// Android
	case strings.Contains(ua, "android"):
		return "Android"
	// Desktop OS
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "linux"), strings.Contains(ua, "x11"):
		return "Linux"
	default:
		return "Unknown"
	}
}
