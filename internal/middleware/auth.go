package middleware

import (
	"context"
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Auth 认证插件
func Auth(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := config.Get()
			// 获取认证头
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				utils.Unauthorized(w)
				return
			}
			// 获取认证信息
			secretID := strings.TrimPrefix(authHeader, "Bearer ")
			timestamp := r.Header.Get("X-Timestamp")
			nonce := r.Header.Get("X-Nonce")
			signature := r.Header.Get("X-Signature")
			// 校验参数是否缺失
			if secretID == "" || timestamp == "" || nonce == "" || signature == "" {
				utils.Unauthorized(w)
				return
			}
			// 校验时间戳
			ts, err := strconv.ParseInt(timestamp, 10, 64)
			if err != nil {
				utils.Unauthorized(w)
				return
			}
			if len(timestamp) == 13 {
				ts /= 1000
			}
			// 校验时间窗口
			if utils.Abs(time.Now().Unix()-ts) > cfg.Auth.TimestampWindow {
				utils.Forbidden(w, "Request expired")
				return
			}
			// nonce 唯一性校验
			nonceKey := cfg.Redis.ProjectPrefix + ":nonce:" + secretID + ":" + nonce
			count, err := app.Redis.IncrAndExpire(nonceKey, time.Duration(cfg.Auth.NonceWindowHour)*time.Hour, false)
			if err != nil {
				utils.InternalServerError(w)
				return
			}
			if count > cfg.Auth.NonceWindow {
				utils.Forbidden(w, "Nonce window exceeded")
				return
			}
			// 查询API密钥信息
			apiKeyInfo, ok, err := app.GetApiKeyInfo(secretID)
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
			// 校验路由权限
			route, ok := r.Context().Value(utils.RouteKey).(*config.Route)
			if !ok {
				utils.NotFound(w)
				return
			}
			if !utils.Contains(apiKeyInfo.RouteIDs, route.ID) {
				utils.Forbidden(w, "No access to this route")
				return
			}
			// 校验签名
			if !utils.Verify(apiKeyInfo.SecretKey, r, timestamp, nonce, signature) {
				utils.Unauthorized(w, "Signature mismatch")
				return
			}
			ctx := context.WithValue(r.Context(), utils.ApiKeyInfoKey, apiKeyInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
