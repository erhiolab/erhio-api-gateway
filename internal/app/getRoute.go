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

// routeG 路由信息缓存
var routeG singleflight.Group

// GetRoute 获取路由信息
func (app *App) GetRoute(routeID int64) (*models.Route, bool, error) {
	if routeID <= 0 {
		return nil, false, errors.New("routeID is invalid")
	}
	cfg := config.Get()
	cacheKey := cfg.Redis.ProjectPrefix + ":route:" + strconv.FormatInt(routeID, 10)
	localCacheExpire := time.Duration(cfg.DatabaseConfig.LocalCacheExpire) * time.Minute
	redisCacheExpire := time.Duration(cfg.DatabaseConfig.RedisCacheExpire) * time.Minute
	redisTTL := redisCacheExpire + time.Duration(rand.Int63n(int64(redisCacheExpire/10)))
	// 先尝试从本地缓存中获取
	if val, ok := app.LocalCache.Get(cacheKey); ok {
		if route, ok := val.(*models.Route); ok {
			return route, true, nil
		}
	}
	v, err, _ := routeG.Do(cacheKey, func() (interface{}, error) {
		// 二次检查本地缓存, 防止并发进入 SingleFlight 后重复逻辑
		if val, ok := app.LocalCache.Get(cacheKey); ok {
			return val, nil
		}
		// 尝试从 Redis 中获取
		route := &models.Route{}
		ok, err := app.Redis.Get(cacheKey, route)
		if err == nil && ok {
			app.LocalCache.Set(cacheKey, route, localCacheExpire, true)
			return route, nil
		} else if err != nil {
			logger.Log.Error("Redis获取路由信息异常", zap.Int64("routeID", routeID), zap.Error(err))
		}
		// 通过数据库获取
		route, ok, err = app.DB.GetRouteByID(routeID)
		if err != nil {
			return nil, err
		}
		if !ok {
			_ = app.Redis.Set(cacheKey, "EMPTY", 5*time.Minute)
			return nil, errors.New("route not found")
		}
		// 写入 Redis
		if err := app.Redis.Set(cacheKey, route, redisTTL); err != nil {
			logger.Log.Warn("写入Redis失败", zap.Error(err))
		}
		// 写入本地缓存
		app.LocalCache.Set(cacheKey, route, localCacheExpire, true)
		return route, nil
	})
	if err != nil {
		return nil, false, err
	}
	return v.(*models.Route), true, nil
}

// MatchRoute 匹配路由
func (app *App) MatchRoute(serviceID int64, path string, method string) (*models.Route, error) {
	routes, err := app.DB.GetAllRoutes()
	if err != nil {
		return nil, err
	}
	for _, route := range routes {
		if route.ServiceID == serviceID && route.Path == path && route.Method == method {
			// 从缓存中获取完整的路由信息
			routeWithDetails, ok, err := app.GetRoute(route.ID)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			return routeWithDetails, nil
		}
	}
	return nil, nil
}

// ClearRouteCache 清除路由缓存
func (app *App) ClearRouteCache(routeID int64) error {
	if routeID <= 0 {
		return errors.New("routeID is invalid")
	}
	cfg := config.Get()
	cacheKey := cfg.Redis.ProjectPrefix + ":route:" + strconv.FormatInt(routeID, 10)
	app.LocalCache.Delete(cacheKey)
	err := app.Redis.Del(cacheKey)
	if err != nil {
		return err
	}
	return nil
}

// ClearAllRouteCache 清除所有路由缓存
func (app *App) ClearAllRouteCache() error {
	cfg := config.Get()
	pattern := cfg.Redis.ProjectPrefix + ":route:*"
	// 通过前缀获取所有路由缓存键
	keys, err := app.Redis.GetKeysByPattern(pattern)
	if err != nil {
		return err
	}
	// 清除每个路由的缓存
	for _, key := range keys {
		app.LocalCache.Delete(key)
		if err := app.Redis.Del(key); err != nil {
			logger.Log.Error("清除路由缓存失败",
				zap.String("key", key),
				zap.Error(err),
			)
		}
	}
	return nil
}
