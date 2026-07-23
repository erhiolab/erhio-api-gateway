package utils

import (
	"elake-api-gateway/internal/models"
	"net/http"
	"strings"
)

// GetUserAgentInfo 获取客户端 User-Agent 信息
func GetUserAgentInfo(r *http.Request) *models.UserAgent {
	ua := r.Header.Get("User-Agent")
	if ua == "" {
		ua = "unknown"
	}
	return &models.UserAgent{
		UserAgent: ua,
		Device:    parseUserAgent(ua),
	}
}

// parseUserAgent 解析客户端 User-Agent
func parseUserAgent(ua string) string {
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
