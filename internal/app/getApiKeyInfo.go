package app

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"errors"
	"math/rand"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

// appG API密钥信息缓存
var appG singleflight.Group

// GetApiKeyInfo 获取API密钥信息
func (app *App) GetApiKeyInfo(secretID string) (*models.APIKeyInfo, bool, error) {
	if secretID == "" {
		return nil, false, errors.New("secretID is empty")
	}
	cfg := config.Get()
	cacheKey := cfg.Redis.ProjectPrefix + ":api_key_info:" + secretID
	localCacheExpire := time.Duration(cfg.Gateway.LocalCacheExpire) * time.Minute
	redisCacheExpire := time.Duration(cfg.Gateway.RedisCacheExpire) * time.Minute
	redisTTL := redisCacheExpire + time.Duration(rand.Int63n(int64(redisCacheExpire/10)))
	// 先尝试从本地缓存中获取
	if val, ok := app.LocalCache.Get(cacheKey); ok {
		if info, ok := val.(*models.APIKeyInfo); ok {
			return info, true, nil
		}
	}
	v, err, _ := appG.Do(cacheKey, func() (interface{}, error) {
		// 二次检查本地缓存, 防止并发进入 SingleFlight 后重复逻辑
		if val, ok := app.LocalCache.Get(cacheKey); ok {
			return val, nil
		}
		apiKeyInfo := &models.APIKeyInfo{}
		// 尝试从 Redis 中获取
		ok, err := app.Redis.Get(cacheKey, apiKeyInfo)
		if err == nil && ok {
			if err := app.decryptAndCacheLocal(cacheKey, apiKeyInfo, localCacheExpire); err != nil {
				return nil, err
			}
			return apiKeyInfo, nil
		} else if err != nil {
			logger.Log.Error("Redis获取API密钥信息异常", zap.String("key", cacheKey), zap.Error(err))
		}
		// 通过数据库获取
		info, ok, err := app.DB.GetApiKeyInfo(secretID)
		if err != nil {
			return nil, err
		}
		if !ok {
			_ = app.Redis.Set(cacheKey, "EMPTY", 5*time.Minute)
			return nil, errors.New("api key not found")
		}
		// 写入 Redis
		if err := app.Redis.Set(cacheKey, info, redisTTL); err != nil {
			logger.Log.Warn("写入Redis失败", zap.Error(err))
		}
		// 解密后写入本地缓存
		if err := app.decryptAndCacheLocal(cacheKey, info, localCacheExpire); err != nil {
			return nil, err
		}
		return info, nil
	})
	if err != nil {
		return nil, false, err
	}
	return v.(*models.APIKeyInfo), true, nil
}

// 解密并存入本地缓存
func (app *App) decryptAndCacheLocal(key string, info *models.APIKeyInfo, expire time.Duration) error {
	sk, err := utils.DecryptSecret(info.SecretKey)
	if err != nil {
		logger.Log.Error("API密钥解密失败", zap.Error(err))
		return err
	}
	info.SecretKey = sk
	app.LocalCache.Set(key, info, expire, true)
	return nil
}

// ExpiredApiKeyInfoCache 让API密钥信息缓存立刻过期
func (app *App) ExpiredApiKeyInfoCache(secretID string) error {
	if secretID == "" {
		return errors.New("secretID is empty")
	}
	cfg := config.Get().Redis
	cacheKey := cfg.ProjectPrefix + ":api_key_info:" + secretID
	app.LocalCache.Delete(cacheKey)
	err := app.Redis.Del(cacheKey)
	if err != nil {
		return err
	}
	return nil
}
