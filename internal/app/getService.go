package app

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"errors"
	"math/rand"
	"path"
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
func (app *App) MatchService(reqPath string) (*models.Service, string, error) {
	// 路径规范化: 清理 // 和 /../ 等, 防止路径绕过
	reqPath = path.Clean(reqPath)
	services, err := app.DB.GetAllServices()
	if err != nil {
		return nil, "", err
	}
	for _, service := range services {
		if matchBasePath(reqPath, service.BasePath) {
			// 从缓存中获取完整的服务信息
			serviceWithNodes, ok, err := app.GetService(service.ID)
			if err != nil {
				return nil, "", err
			}
			if !ok {
				continue
			}
			newPath := reqPath
			if service.BasePath != "/" && service.BasePath != "" {
				newPath = reqPath[len(service.BasePath):]
				if newPath == "" {
					newPath = "/"
				}
			}
			return serviceWithNodes, newPath, nil
		}
	}
	return nil, "", nil
}

// matchBasePath 判断请求路径是否匹配服务的 BasePath
func matchBasePath(reqPath, basePath string) bool {
	// 空 BasePath 不匹配任何路径
	if basePath == "" {
		return false
	}
	// "/" 匹配所有路径
	if basePath == "/" {
		return true
	}
	// 精确匹配
	if reqPath == basePath {
		return true
	}
	// 前缀匹配: 路径以 BasePath 开头且下一个字符是 '/'
	if len(reqPath) > len(basePath) &&
		reqPath[:len(basePath)] == basePath && reqPath[len(basePath)] == '/' {
		return true
	}
	return false
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
	cfg := config.Get()
	pattern := cfg.Redis.ProjectPrefix + ":service:*"
	// 通过前缀获取所有服务缓存键
	keys, err := app.Redis.GetKeysByPattern(pattern)
	if err != nil {
		return err
	}
	// 清除每个服务的缓存
	for _, key := range keys {
		app.LocalCache.Delete(key)
		if err := app.Redis.Del(key); err != nil {
			logger.Log.Error("清除服务缓存失败",
				zap.String("key", key),
				zap.Error(err),
			)
		}
	}
	return nil
}
