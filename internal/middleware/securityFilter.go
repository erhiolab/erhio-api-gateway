package middleware

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
)

// suspiciousFilter 可疑路径
var suspiciousPathPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\.php$`),
	regexp.MustCompile(`(?i)\.asp$`),
	regexp.MustCompile(`(?i)\.aspx$`),
	regexp.MustCompile(`(?i)\.jsp$`),

	regexp.MustCompile(`(?i)^/wp-`),
	regexp.MustCompile(`(?i)^/wordpress`),

	regexp.MustCompile(`(?i)^/\.env`),
	regexp.MustCompile(`(?i)^/\.git`),

	regexp.MustCompile(`(?i)/vendor/phpunit`),

	regexp.MustCompile(`(?i)^/cgi-bin`),

	regexp.MustCompile(`(?i)/actuator`),

	regexp.MustCompile(`(?i)/server-status`),
}

// SecurityFilter 安全过滤插件
func SecurityFilter(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			cfg := config.Get()
			ip := utils.GetClientIP(r)
			// 检查IP是否被封禁
			banKey := cfg.Redis.ProjectPrefix + ":security:ban:" + ip
			var banned string
			if found, _ := app.Redis.Get(banKey, &banned); found {
				utils.Forbidden(w, "IP 已被封禁")
				return
			}
			path := r.URL.Path
			if isSuspiciousRequest(path) {
				// 1小时计数
				hourlyKey := cfg.Redis.ProjectPrefix + ":security:hourly:" + ip
				hourlyCount, _ := app.Redis.IncrAndExpire(hourlyKey, time.Hour, false)
				// 1天计数
				dailyKey := cfg.Redis.ProjectPrefix + ":security:daily:" + ip
				dailyCount, _ := app.Redis.IncrAndExpire(dailyKey, 24*time.Hour, false)
				// 判断封禁级别
				if dailyCount >= 20 {
					// 1天内超过20次, 永久封禁
					blacklistID := fmt.Sprintf("auto-ban-ip-%s-%d", ip, time.Now().UnixNano())
					blacklist := &models.Blacklist{
						ID:          blacklistID,
						Type:        "ip",
						Value:       ip,
						Description: fmt.Sprintf("安全过滤自动封禁: 24小时内可疑请求%d次", dailyCount),
					}
					if err := app.DB.InsertBlacklist(blacklist); err != nil {
						logger.WithRequestLogCtx(ctx, r).Error("安全过滤插件: 插入IP黑名单失败",
							zap.String("ip", ip),
							zap.Error(err),
						)
					} else {
						// 清除黑名单缓存,使新插入的IP立即生效
						_ = app.ClearBlacklistCache("ip")
						logger.WithRequestLogCtx(ctx, r).Warn("安全过滤插件: IP永久封禁(已插入数据库)",
							zap.String("ip", ip),
							zap.Int64("dailyCount", dailyCount),
						)
					}
					utils.Forbidden(w, "IP 已被封禁")
					return
				}
				if hourlyCount >= 10 {
					// 1小时内超过10次, 封禁1天
					_ = app.Redis.Set(banKey, "1", 24*time.Hour)
					logger.WithRequestLogCtx(ctx, r).Warn("安全过滤插件: IP封禁1天",
						zap.String("ip", ip),
						zap.Int64("hourlyCount", hourlyCount),
					)
					utils.Forbidden(w, "IP 已被封禁")
					return
				}
				if hourlyCount >= 3 {
					// 1小时内超过3次, 封禁1小时
					_ = app.Redis.Set(banKey, "1", time.Hour)
					logger.WithRequestLogCtx(ctx, r).Warn("安全过滤插件: IP封禁1小时",
						zap.String("ip", ip),
						zap.Int64("hourlyCount", hourlyCount),
					)
					utils.Forbidden(w, "IP 已被封禁")
					return
				}
				logger.WithRequestLogCtx(ctx, r).Warn("安全过滤插件: 可疑请求",
					zap.String("ip", ip),
					zap.Int64("hourlyCount", hourlyCount),
					zap.Int64("dailyCount", dailyCount),
				)
				utils.NotFound(w, "资源不存在")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isSuspiciousRequest 是否为可疑请求
func isSuspiciousRequest(path string) bool {
	if strings.Contains(path, "..") {
		return true
	}
	for _, reg := range suspiciousPathPatterns {
		if reg.MatchString(path) {
			return true
		}
	}
	return false
}
