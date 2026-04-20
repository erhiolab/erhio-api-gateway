package middleware

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// localRateLimiter QPS本地限流器
type localRateLimiter struct {
	mu       sync.RWMutex
	counters map[string]*slidingWindowCounter
}

// slidingWindowCounter 滑动窗口计数器
type slidingWindowCounter struct {
	windowStart time.Time
	count       int64
}

// globalRateLimiter 全局 QPS 本地限流器
var globalRateLimiter = &localRateLimiter{
	counters: make(map[string]*slidingWindowCounter),
}

// checkQPS 检查 QPS 限流(本地滑动窗口)
func (l *localRateLimiter) checkQPS(key string, limit int64) bool {
	if limit <= 0 {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	counter, ok := l.counters[key]
	// 如果是新的一秒, 重置窗口
	if !ok || now.Sub(counter.windowStart) >= time.Second {
		l.counters[key] = &slidingWindowCounter{
			windowStart: now,
			count:       1,
		}
		return true
	}
	if counter.count >= limit {
		return false
	}
	counter.count++
	return true
}

// tokenBucket 令牌桶
type tokenBucket struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
}

// bucket 令牌桶
type bucket struct {
	tokens     float64
	lastRefill time.Time
}

// globalTokenBucket 全局令牌桶
var globalTokenBucket = &tokenBucket{
	buckets: make(map[string]*bucket),
}

// checkQPM 检查 QPM 限流(本地令牌桶)
func (tb *tokenBucket) checkQPM(key string, limit int64) bool {
	if limit <= 0 {
		return true
	}

	// 每分钟补充 limit 个, 每秒补充 limit/60.0 个
	refillRate := float64(limit) / 60.0
	now := time.Now()

	tb.mu.Lock()
	defer tb.mu.Unlock()

	b, ok := tb.buckets[key]
	if !ok {
		b = &bucket{
			tokens:     float64(limit),
			lastRefill: now,
		}
		tb.buckets[key] = b
	}

	// 补充令牌
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * refillRate
	if b.tokens > float64(limit) {
		b.tokens = float64(limit)
	}
	b.lastRefill = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// init 初始化限流器
func init() {
	// 定期清理过期的本地缓存, 防止内存泄漏
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			now := time.Now()
			// 清理 QPS
			globalRateLimiter.mu.Lock()
			for k, v := range globalRateLimiter.counters {
				if now.Sub(v.windowStart) > 10*time.Second {
					delete(globalRateLimiter.counters, k)
				}
			}
			globalRateLimiter.mu.Unlock()
			// 清理 QPM
			globalTokenBucket.mu.Lock()
			for k, v := range globalTokenBucket.buckets {
				if now.Sub(v.lastRefill) > 5*time.Minute {
					delete(globalTokenBucket.buckets, k)
				}
			}
			globalTokenBucket.mu.Unlock()
		}
	}()
}

// RateLimit 限流器插件
func RateLimit(app *app.App) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			cfg := config.Get()
			route, ok := ctx.Value(utils.RouteKey).(*models.Route)
			if !ok {
				logger.WithRequestLogCtx(ctx).Warn("限流器插件: 路由不存在",
					zap.Int64("route_id", route.ID),
					zap.String("path", route.Path),
					zap.String("method", route.Method),
				)
				utils.NotFound(w)
				return
			}
			var qpmKey, qpsKey string
			var qpmLimit, qpsLimit int64
			apiKeyInfo := models.APIKeyInfo{}
			if apiKeyInfoPtr, ok := ctx.Value(utils.ApiKeyInfoKey).(*models.APIKeyInfo); ok && apiKeyInfoPtr != nil {
				apiKeyInfo = *apiKeyInfoPtr
			}
			ipLoc, ok := ctx.Value(utils.ClientIPKey).(*models.IPLocation)
			if !ok || ipLoc == nil {
				logger.WithRequestLogCtx(ctx).Warn("限流器插件: 客户端IP不存在")
				utils.BadRequest(w, "empty ip")
				return
			}
			if apiKeyInfo.SecretID != "" {
				// 按 API Key 限流
				prefix := cfg.Redis.ProjectPrefix + ":limit:key:" + apiKeyInfo.SecretID
				qpmKey = prefix + ":qpm"
				qpsKey = prefix + ":qps"
				qpmLimit = apiKeyInfo.QPM
				qpsLimit = apiKeyInfo.QPS
			} else if ipLoc != nil {
				// 按 IP 限流
				prefix := cfg.Redis.ProjectPrefix + ":limit:ip:" + ipLoc.IP
				qpmKey = prefix + ":qpm"
				qpsKey = prefix + ":qps"
				// 如果路由没配置限流, 则使用全局配置
				qpmLimit = route.QPM
				if qpmLimit <= 0 {
					qpmLimit = cfg.DatabaseConfig.Auth.QpmLimit
				}
				qpsLimit = route.QPS
				if qpsLimit <= 0 {
					qpsLimit = cfg.DatabaseConfig.Auth.QpsLimit
				}
			}
			// 执行 QPS 限流 (本地滑动窗口 + 异步同步 Redis)
			if qpsLimit > 0 {
				if !globalRateLimiter.checkQPS(qpsKey, qpsLimit) {
					logger.WithRequestLogCtx(ctx).Warn("限流器插件: QPS限制超出",
						zap.String("key", qpsKey),
						zap.Int64("limit", qpsLimit),
					)
					utils.TooManyRequests(w)
					return
				}
				// 异步同步 QPS 到 Redis
				go func(k string) {
					_, err := app.Redis.IncrAndExpire(k, time.Second, false)
					if err != nil {
						logger.Log.Debug("限流器插件: 同步 QPS 到 Redis 失败", zap.Error(err))
					}
				}(qpsKey)
			}
			// 执行 QPM 限流 (本地桶 + 异步同步 Redis)
			if qpmLimit > 0 {
				if !globalTokenBucket.checkQPM(qpmKey, qpmLimit) {
					logger.WithRequestLogCtx(ctx).Warn("限流器插件: QPM限制超出",
						zap.String("key", qpmKey),
						zap.Int64("limit", qpmLimit),
					)
					utils.TooManyRequests(w)
					return
				}
				// 异步同步 QPM 到 Redis
				go func(k string) {
					_, err := app.Redis.IncrAndExpire(k, time.Minute, false)
					if err != nil {
						logger.Log.Debug("限流器插件: 同步 QPM 到 Redis 失败",
							zap.Error(err),
						)
					}
				}(qpmKey)
			}
			next.ServeHTTP(w, r)
		})
	}
}
