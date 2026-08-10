package app

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"errors"
	"math/rand"
	"strconv"
	"strings"
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
func (app *App) MatchRoute(serviceID int64, path string, method string) (*models.Route, map[string]string, error) {
	routes, err := app.DB.GetAllRoutes()
	if err != nil {
		return nil, nil, err
	}
	var bestMatch *models.Route
	var bestParams map[string]string
	for i := range routes {
		route := &routes[i]
		if route.ServiceID != serviceID || route.Method != method {
			continue
		}
		params, ok := matchPath(route.Path, path)
		if !ok {
			continue
		}
		if len(params) == 0 {
			bestMatch = route
			bestParams = params
			break
		}
		if bestMatch == nil {
			bestMatch = route
			bestParams = params
		}
	}
	if bestMatch == nil {
		return nil, nil, nil
	}
	routeWithDetails, ok, err := app.GetRoute(bestMatch.ID)
	if err != nil || !ok {
		return nil, nil, err
	}
	return routeWithDetails, bestParams, nil
}

// matchPath 匹配路径模式
func matchPath(pattern, path string) (map[string]string, bool) {
	if pattern == path {
		return map[string]string{}, true
	}
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(patternParts) != len(pathParts) {
		return nil, false
	}
	params := make(map[string]string)
	for i, pp := range patternParts {
		if strings.HasPrefix(pp, ":") {
			params[pp[1:]] = pathParts[i]
		} else if pp != pathParts[i] {
			return nil, false
		}
	}
	return params, true
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
