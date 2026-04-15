package repository

import (
	"context"
	"elake-api-gateway/internal/config"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisManager 封装 Redis 操作
type RedisManager struct {
	client *redis.Client
}

// NewRedisManager 初始化管理器
func NewRedisManager(client *redis.Client) *RedisManager {
	return &RedisManager{client: client}
}

// Get 获取并反序列化对象 [泛型支持]
func (r *RedisManager) Get(key string, dest interface{}) (bool, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Redis.ReadTimeout)*time.Second)
	defer cancel()
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	// 特殊状态: 明确告知不存在
	if val == "EMPTY" {
		return false, nil
	}
	// 将字符串反序列化为对象
	// 如果 dest 是 string 类型就直接赋值
	if s, ok := dest.(*string); ok {
		*s = val
		return true, nil
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, err
	}
	return true, nil
}

// Set 设置键值对
func (r *RedisManager) Set(key string, value interface{}, expiration time.Duration) error {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Redis.WriteTimeout)*time.Second)
	defer cancel()
	var data interface{}
	switch v := value.(type) {
	case string, int, int64, float64, bool:
		data = v
	default:
		// 结构体或 Map 自动转为 JSON
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		data = string(b)
	}
	return r.client.Set(ctx, key, data, expiration).Err()
}

// Del 删除键
func (r *RedisManager) Del(key string) error {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Redis.WriteTimeout)*time.Second)
	defer cancel()
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return err
	}
	return nil
}

// Expire 修改过期时间
func (r *RedisManager) Expire(key string, expiration time.Duration) (bool, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Redis.WriteTimeout)*time.Second)
	defer cancel()
	success, err := r.client.Expire(ctx, key, expiration).Result()
	if err != nil {
		return false, err
	}
	return success, nil
}

// IncrAndExpire 增加计数并设置过期时间
func (r *RedisManager) IncrAndExpire(key string, expiration time.Duration, refresh bool) (int64, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Redis.WriteTimeout)*time.Second)
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
		return 0, err
	}
	return val.(int64), nil
}

// Exists 检查是否存在
func (r *RedisManager) Exists(key string) (bool, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Redis.ReadTimeout)*time.Second)
	defer cancel()
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetKeysByPattern 通过模式匹配获取所有键
func (r *RedisManager) GetKeysByPattern(pattern string) ([]string, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Redis.ReadTimeout)*time.Second)
	defer cancel()
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// Close 关闭 Redis 连接
func (r *RedisManager) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}
