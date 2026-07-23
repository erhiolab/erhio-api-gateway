package app

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"math/rand"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

// blacklistG 黑名单缓存
var blacklistG singleflight.Group

// GetBlacklistByType 根据类型获取黑名单列表
func (app *App) GetBlacklistByType(blacklistType string) ([]models.Blacklist, error) {
	if blacklistType == "" {
		return []models.Blacklist{}, nil
	}
	cfg := config.Get()
	cacheKey := cfg.Redis.ProjectPrefix + ":blacklist:" + blacklistType
	localCacheExpire := time.Duration(cfg.DatabaseConfig.LocalCacheExpire) * time.Minute
	redisCacheExpire := time.Duration(cfg.DatabaseConfig.RedisCacheExpire) * time.Minute
	redisTTL := redisCacheExpire + time.Duration(rand.Int63n(int64(redisCacheExpire/10)))
	// 先尝试从本地缓存中获取
	if val, ok := app.LocalCache.Get(cacheKey); ok {
		if blacklists, ok := val.([]models.Blacklist); ok {
			return blacklists, nil
		}
	}
	v, err, _ := blacklistG.Do(cacheKey, func() (interface{}, error) {
		// 二次检查本地缓存, 防止并发进入 SingleFlight 后重复逻辑
		if val, ok := app.LocalCache.Get(cacheKey); ok {
			return val, nil
		}
		// 尝试从 Redis 中获取
		var blacklists []models.Blacklist
		ok, err := app.Redis.Get(cacheKey, &blacklists)
		if err == nil && ok {
			app.LocalCache.Set(cacheKey, blacklists, localCacheExpire, true)
			return blacklists, nil
		} else if err != nil {
			logger.Log.Error("Redis获取黑名单信息异常", zap.String("type", blacklistType), zap.Error(err))
		}
		// 通过数据库获取
		blacklists, err = app.DB.GetBlacklistByType(blacklistType)
		if err != nil {
			return nil, err
		}
		// 写入 Redis
		if err := app.Redis.Set(cacheKey, blacklists, redisTTL); err != nil {
			logger.Log.Warn("写入Redis失败", zap.Error(err))
		}
		// 写入本地缓存
		app.LocalCache.Set(cacheKey, blacklists, localCacheExpire, true)
		return blacklists, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]models.Blacklist), nil
}

// GetAllBlacklist 获取所有黑名单
func (app *App) GetAllBlacklist() ([]models.Blacklist, error) {
	cfg := config.Get()
	cacheKey := cfg.Redis.ProjectPrefix + ":blacklist:all"
	localCacheExpire := time.Duration(cfg.DatabaseConfig.LocalCacheExpire) * time.Minute
	redisCacheExpire := time.Duration(cfg.DatabaseConfig.RedisCacheExpire) * time.Minute
	redisTTL := redisCacheExpire + time.Duration(rand.Int63n(int64(redisCacheExpire/10)))
	// 先尝试从本地缓存中获取
	if val, ok := app.LocalCache.Get(cacheKey); ok {
		if blacklists, ok := val.([]models.Blacklist); ok {
			return blacklists, nil
		}
	}
	v, err, _ := blacklistG.Do(cacheKey, func() (interface{}, error) {
		// 二次检查本地缓存, 防止并发进入 SingleFlight 后重复逻辑
		if val, ok := app.LocalCache.Get(cacheKey); ok {
			return val, nil
		}
		// 尝试从 Redis 中获取
		var blacklists []models.Blacklist
		ok, err := app.Redis.Get(cacheKey, &blacklists)
		if err == nil && ok {
			app.LocalCache.Set(cacheKey, blacklists, localCacheExpire, true)
			return blacklists, nil
		} else if err != nil {
			logger.Log.Error("Redis获取所有黑名单信息异常", zap.Error(err))
		}
		// 通过数据库获取
		blacklists, err = app.DB.GetAllBlacklist()
		if err != nil {
			return nil, err
		}
		// 写入 Redis
		if err := app.Redis.Set(cacheKey, blacklists, redisTTL); err != nil {
			logger.Log.Warn("写入Redis失败", zap.Error(err))
		}
		// 写入本地缓存
		app.LocalCache.Set(cacheKey, blacklists, localCacheExpire, true)
		return blacklists, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]models.Blacklist), nil
}

// ClearBlacklistCache 清除指定类型黑名单缓存
func (app *App) ClearBlacklistCache(blacklistType string) error {
	if blacklistType == "" {
		return nil
	}
	cfg := config.Get().Redis
	cacheKey := cfg.ProjectPrefix + ":blacklist:" + blacklistType
	app.LocalCache.Delete(cacheKey)
	err := app.Redis.Del(cacheKey)
	if err != nil {
		return err
	}
	return nil
}

// ClearAllBlacklistCache 清除所有黑名单缓存
func (app *App) ClearAllBlacklistCache() error {
	cfg := config.Get().Redis
	pattern := cfg.ProjectPrefix + ":blacklist:*"
	keys, err := app.Redis.GetKeysByPattern(pattern)
	if err != nil {
		return err
	}
	for _, key := range keys {
		app.LocalCache.Delete(key)
		if err := app.Redis.Del(key); err != nil {
			logger.Log.Error("清除黑名单缓存失败",
				zap.String("key", key),
				zap.Error(err),
			)
		}
	}
	return nil
}
