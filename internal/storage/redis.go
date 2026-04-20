package storage

import (
	"context"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/service/healthManager"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// InitRedis 初始化 Redis 连接
func InitRedis() *redis.Client {
	cfg := config.Get()
	addr := fmt.Sprintf("%s:%d",
		cfg.Redis.Host,
		cfg.Redis.Port,
	)
	rdb := redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        cfg.Redis.Password,
		DB:              cfg.Redis.DB,
		PoolSize:        cfg.Redis.PoolSize,
		MinIdleConns:    cfg.Redis.MinIdleConnections,
		ConnMaxIdleTime: time.Duration(cfg.Redis.ConnectionMaxIdleTime) * time.Minute,
		DialTimeout:     time.Duration(cfg.Redis.DialTimeout) * time.Second,
		ReadTimeout:     time.Duration(cfg.Redis.ReadTimeout) * time.Second,
		WriteTimeout:    time.Duration(cfg.Redis.WriteTimeout) * time.Second,
		ConnMaxLifetime: time.Duration(cfg.Redis.ConnMaxLifetime) * time.Minute,
		PoolTimeout:     time.Duration(cfg.Redis.PoolTimeout) * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Log.Fatal("Redis 连接失败:", zap.Error(err))
	}
	logger.Log.Info("Redis 连接成功")
	healthManager.Global().Register(
		"Redis",
		cfg.Health.RedisHealthCheckFailThreshold,
		cfg.Health.RedisHealthCheckOKThreshold,
	)
	// 健康检查
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.Health.RedisHealthCheckInterval) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := rdb.Ping(ctx).Err()
			cancel()
			healthManager.Global().Report("Redis", err)
		}
	}()
	return rdb
}
