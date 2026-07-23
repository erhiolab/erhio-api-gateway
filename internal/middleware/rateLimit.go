package middleware

import (
	"context"
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"hash/fnv"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

const shardCount = 64

// localRateLimiter QPS本地限流器
type localRateLimiter struct {
	shards [shardCount]struct {
		mu       sync.Mutex
		counters map[string]*slidingWindowCounter
	}
}

// slidingWindowCounter 滑动窗口计数器
type slidingWindowCounter struct {
	windowStart time.Time
	count       int64
}

// globalRateLimiter 全局 QPS 本地限流器
var globalRateLimiter = &localRateLimiter{}

// init 初始化 QPS 本地限流器的计数器映射
func init() {
	for i := range globalRateLimiter.shards {
		globalRateLimiter.shards[i].counters = make(map[string]*slidingWindowCounter)
	}
}

// getShard 获取 QPS 本地限流器的分片索引
func (l *localRateLimiter) getShard(key string) int {
	h := fnv.New32a()
	_, err := h.Write([]byte(key))
	if err != nil {
		return 0
	}
	return int(h.Sum32()) % shardCount
}

// checkQPS 检查 QPS 本地限流器是否超过限制
func (l *localRateLimiter) checkQPS(key string, limit int64) bool {
	if limit <= 0 {
		return true
	}
	now := time.Now()
	s := &l.shards[l.getShard(key)]
	s.mu.Lock()
	defer s.mu.Unlock()
	counter, ok := s.counters[key]
	if !ok || now.Sub(counter.windowStart) >= time.Second {
		s.counters[key] = &slidingWindowCounter{
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

// tokenBucket 令牌桶限流器
type tokenBucket struct {
	shards [shardCount]struct {
		mu      sync.Mutex
		buckets map[string]*bucket
	}
}

// bucket 令牌桶
type bucket struct {
	tokens     float64
	lastRefill time.Time
}

// globalTokenBucket 全局令牌桶限流器
var globalTokenBucket = &tokenBucket{}

// init 初始化令牌桶限流器的令牌桶映射
func init() {
	for i := range globalTokenBucket.shards {
		globalTokenBucket.shards[i].buckets = make(map[string]*bucket)
	}
}

// getShard 获取令牌桶限流器的分片索引
func (tb *tokenBucket) getShard(key string) int {
	h := fnv.New32a()
	_, err := h.Write([]byte(key))
	if err != nil {
		return 0
	}
	return int(h.Sum32()) % shardCount
}

// checkQPM 检查令牌桶限流器是否超过限制
func (tb *tokenBucket) checkQPM(key string, limit int64) bool {
	if limit <= 0 {
		return true
	}
	refillRate := float64(limit) / 60.0
	now := time.Now()
	s := &tb.shards[tb.getShard(key)]
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.buckets[key]
	if !ok {
		b = &bucket{
			tokens:     float64(limit),
			lastRefill: now,
		}
		s.buckets[key] = b
	}
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

// syncItem 同步项
type syncItem struct {
	key string
	ttl time.Duration
}

// syncCh 同步通道
var syncCh = make(chan syncItem, 65536)

// init 初始化令牌桶限流器的同步通道
func init() {
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		batch := make(map[string]time.Duration)
		for {
			select {
			case item := <-syncCh:
				batch[item.key] = item.ttl
			case <-ticker.C:
				if len(batch) == 0 {
					continue
				}
				flushBatch(batch)
				batch = make(map[string]time.Duration)
			}
		}
	}()
}

// flushBatch 批量同步令牌桶限流器的令牌桶到 Redis
func flushBatch(batch map[string]time.Duration) {
	redisClient := app.GetRedisClient()
	if redisClient == nil {
		return
	}
	pipe := redisClient.Pipeline()
	for k, ttl := range batch {
		pipe.Incr(context.Background(), k)
		pipe.Expire(context.Background(), k, ttl)
	}
	if _, err := pipe.Exec(context.Background()); err != nil {
		logger.Log.Debug("限流器插件: 批量同步 Redis 失败",
			zap.Error(err),
		)
	}
}

// init 初始化令牌桶限流器的定时任务
func init() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			now := time.Now()
			for i := range globalRateLimiter.shards {
				s := &globalRateLimiter.shards[i]
				s.mu.Lock()
				for k, v := range s.counters {
					if now.Sub(v.windowStart) > 10*time.Second {
						delete(s.counters, k)
					}
				}
				s.mu.Unlock()
			}
			for i := range globalTokenBucket.shards {
				s := &globalTokenBucket.shards[i]
				s.mu.Lock()
				for k, v := range s.buckets {
					if now.Sub(v.lastRefill) > 5*time.Minute {
						delete(s.buckets, k)
					}
				}
				s.mu.Unlock()
			}
		}
	}()
}

// RateLimit 令牌桶限流中间件
func RateLimit() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			cfg := config.Get()
			route, ok := ctx.Value(utils.RouteKey).(*models.Route)
			if !ok {
				logger.WithRequestLogCtx(ctx, r).Warn("限流器插件: 路由不存在",
					zap.Int64("route_id", route.ID),
				)
				utils.NotFound(w, "路由不存在")
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
				logger.WithRequestLogCtx(ctx, r).Warn("限流器插件: 客户端IP不存在")
				utils.BadRequest(w, "客户端IP为空")
				return
			}
			if apiKeyInfo.SecretID != "" {
				prefix := cfg.Redis.ProjectPrefix + ":limit:key:" + apiKeyInfo.SecretID
				qpmKey = prefix + ":qpm"
				qpsKey = prefix + ":qps"
				qpmLimit = apiKeyInfo.QPM
				qpsLimit = apiKeyInfo.QPS
			} else if ipLoc != nil {
				prefix := cfg.Redis.ProjectPrefix + ":limit:ip:" + ipLoc.IP
				qpmKey = prefix + ":qpm"
				qpsKey = prefix + ":qps"
				qpmLimit = route.QPM
				if qpmLimit <= 0 {
					qpmLimit = cfg.DatabaseConfig.Auth.QpmLimit
				}
				qpsLimit = route.QPS
				if qpsLimit <= 0 {
					qpsLimit = cfg.DatabaseConfig.Auth.QpsLimit
				}
			}
			if qpsLimit > 0 {
				if !globalRateLimiter.checkQPS(qpsKey, qpsLimit) {
					logger.WithRequestLogCtx(ctx, r).Warn("限流器插件: QPS限制超出",
						zap.String("key", qpsKey),
						zap.Int64("limit", qpsLimit),
					)
					utils.TooManyRequests(w, "在一秒内请求次数超出限制")
					return
				}
				select {
				case syncCh <- syncItem{key: qpsKey, ttl: time.Second}:
				default:
				}
			}
			if qpmLimit > 0 {
				if !globalTokenBucket.checkQPM(qpmKey, qpmLimit) {
					logger.WithRequestLogCtx(ctx, r).Warn("限流器插件: QPM限制超出",
						zap.String("key", qpmKey),
						zap.Int64("limit", qpmLimit),
					)
					utils.TooManyRequests(w, "在一分钟内请求次数超出限制")
					return
				}
				select {
				case syncCh <- syncItem{key: qpmKey, ttl: time.Minute}:
				default:
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
