package app

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"errors"
	"math/rand"
	"strconv"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

// serviceG 服务信息缓存
var serviceG singleflight.Group

// GetService 获取服务信息
func (app *App) GetService(serviceID int64) (*models.Service, bool, error) {
	if serviceID <= 0 {
		return nil, false, errors.New("serviceID is invalid")
	}
	cfg := config.Get()
	cacheKey := cfg.Redis.ProjectPrefix + ":service:" + strconv.FormatInt(serviceID, 10)
	localCacheExpire := time.Duration(cfg.DatabaseConfig.LocalCacheExpire) * time.Minute
	redisCacheExpire := time.Duration(cfg.DatabaseConfig.RedisCacheExpire) * time.Minute
	redisTTL := redisCacheExpire + time.Duration(rand.Int63n(int64(redisCacheExpire/10)))
	// 先尝试从本地缓存中获取
	if val, ok := app.LocalCache.Get(cacheKey); ok {
		if service, ok := val.(*models.Service); ok {
			return service, true, nil
		}
	}
	v, err, _ := serviceG.Do(cacheKey, func() (interface{}, error) {
		// 二次检查本地缓存, 防止并发进入 SingleFlight 后重复逻辑
		if val, ok := app.LocalCache.Get(cacheKey); ok {
			return val, nil
		}
		service := &models.Service{}
		// 尝试从 Redis 中获取
		ok, err := app.Redis.Get(cacheKey, service)
		if err == nil && ok {
			app.LocalCache.Set(cacheKey, service, localCacheExpire, true)
			return service, nil
		} else if err != nil {
			logger.Log.Error("Redis获取服务信息异常", zap.Int64("serviceID", serviceID), zap.Error(err))
		}
		// 通过数据库获取
		service, ok, err = app.DB.GetServiceWithNodes(serviceID)
		if err != nil {
			return nil, err
		}
		if !ok {
			_ = app.Redis.Set(cacheKey, "EMPTY", 5*time.Minute)
			return nil, errors.New("service not found")
		}
		// 写入 Redis
		if err := app.Redis.Set(cacheKey, service, redisTTL); err != nil {
			logger.Log.Warn("写入Redis失败", zap.Error(err))
		}
		// 写入本地缓存
		app.LocalCache.Set(cacheKey, service, localCacheExpire, true)
		return service, nil
	})
	if err != nil {
		return nil, false, err
	}
	return v.(*models.Service), true, nil
}

// MatchService 匹配服务
func (app *App) MatchService(path string) (*models.Service, string, error) {
	services, err := app.DB.GetAllServices()
	if err != nil {
		return nil, "", err
	}
	for _, service := range services {
		if path == service.BasePath || len(service.BasePath) > 0 && len(path) > len(service.BasePath) && path[:len(service.BasePath)] == service.BasePath && path[len(service.BasePath)] == '/' {
			// 从缓存中获取完整的服务信息
			serviceWithNodes, ok, err := app.GetService(service.ID)
			if err != nil {
				return nil, "", err
			}
			if !ok {
				continue
			}
			newPath := path
			if service.BasePath != "/" {
				newPath = path[len(service.BasePath):]
				if newPath == "" {
					newPath = "/"
				}
			}
			return serviceWithNodes, newPath, nil
		}
	}
	return nil, "", nil
}

// ClearServiceCache 清除服务缓存
func (app *App) ClearServiceCache(serviceID int64) error {
	if serviceID <= 0 {
		return errors.New("serviceID is invalid")
	}
	cfg := config.Get()
	cacheKey := cfg.Redis.ProjectPrefix + ":service:" + strconv.FormatInt(serviceID, 10)
	app.LocalCache.Delete(cacheKey)
	err := app.Redis.Del(cacheKey)
	if err != nil {
		return err
	}
	return nil
}

// ClearAllServiceCache 清除所有服务缓存
func (app *App) ClearAllServiceCache() error {
	// 从数据库获取所有服务ID
	services, err := app.DB.GetAllServices()
	if err != nil {
		return err
	}
	// 清除每个服务的缓存
	for _, service := range services {
		if err := app.ClearServiceCache(service.ID); err != nil {
			logger.Log.Error("清除服务缓存失败", zap.Int64("serviceID", service.ID), zap.Error(err))
		}
	}
	return nil
}
