package repository

import (
	"context"
	"elake-api-gateway/internal/logger"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisManager 封装 Redis 操作
type RedisManager struct {
	client *redis.Client
}

// NewRedisManager 初始化管理器
func NewRedisManager(client *redis.Client) *RedisManager {
	return &RedisManager{client: client}
}

// DefaultTimeout 默认操作超时时间
const DefaultTimeout = 2 * time.Second

// Get 获取字符串值 (结果, 是否存在, 错误)
func (r *RedisManager) Get(key string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", false, nil
		}
		logger.Log.Error("Redis Get 失败", zap.String("key", key), zap.Error(err))
		return "", false, err
	}
	return val, true, nil
}

// Set 设置键值对
func (r *RedisManager) Set(key string, value interface{}, expiration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	err := r.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		logger.Log.Error("Redis Set 失败",
			zap.String("key", key),
			zap.Any("value", value),
			zap.Error(err))
		return err
	}
	return nil
}

// Del 删除键
func (r *RedisManager) Del(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		logger.Log.Error("Redis Del 失败", zap.String("key", key), zap.Error(err))
		return err
	}
	return nil
}

// Expire 修改过期时间 (是否成功, 错误)
func (r *RedisManager) Expire(key string, expiration time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	success, err := r.client.Expire(ctx, key, expiration).Result()
	if err != nil {
		logger.Log.Error("Redis Expire 失败", zap.String("key", key), zap.Error(err))
		return false, err
	}
	return success, nil
}

// IncrAndExpire 增加计数并设置过期时间 (当前计数值, 错误)
func (r *RedisManager) IncrAndExpire(key string, expiration time.Duration, refresh bool) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	// 如果是第一次增加(结果为1), 则设置过期时间
	const script = `
        local current = redis.call("INCR", KEYS[1])
        local refresh = ARGV[2]
        if refresh == "1" or current == 1 then
            redis.call("EXPIRE", KEYS[1], ARGV[1])
        end
        return current
    `
	refreshArg := "0"
	if refresh {
		refreshArg = "1"
	}
	val, err := r.client.Eval(ctx, script, []string{key}, int(expiration.Seconds()), refreshArg).Result()
	if err != nil {
		logger.Log.Error("Redis IncrAndExpire 失败", zap.String("key", key), zap.Error(err))
		return 0, err
	}
	return val.(int64), nil
}

// Exists 检查是否存在 (是否存在, 错误)
func (r *RedisManager) Exists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		logger.Log.Error("Redis Exists 失败", zap.String("key", key), zap.Error(err))
		return false, err
	}
	return count > 0, nil
}

// Close 关闭 Redis 连接
func (r *RedisManager) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}
