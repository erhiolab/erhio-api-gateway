package middleware

import (
	"context"
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// GetRealIP IP解析插件
func GetRealIP(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ip := GetClientIP(r)
			if ip == "" {
				logger.WithRequestLogCtx(ctx).Warn("IP解析插件: 客户端IP为空")
				utils.BadRequest(w, "empty ip")
				return
			}
			rec, err := app.IPDB.GetAll(ip)
			if err != nil {
				logger.WithRequestLogCtx(ctx).Error("IP解析插件: 解析失败",
					zap.String("ip", ip),
					zap.Error(err),
				)
				utils.InternalServerError(w)
				return
			}
			if rec == nil {
				logger.WithRequestLogCtx(ctx).Warn("IP解析插件: 解析失败: 未找到IP信息",
					zap.String("ip", ip),
				)
				utils.BadRequest(w, "empty ip")
				return
			}
			var location = &models.IPLocation{
				IP:           ip,
				CountryShort: rec.CountryShort,
				CountryLong:  rec.CountryLong,
				Region:       rec.Region,
				City:         rec.City,
				Latitude:     rec.Latitude,
				Longitude:    rec.Longitude,
				Zipcode:      rec.Zipcode,
				Timezone:     rec.Timezone,
			}
			ctx = context.WithValue(ctx, utils.ClientIPKey, location)
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
