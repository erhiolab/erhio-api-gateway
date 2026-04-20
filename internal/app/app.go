package app

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/repository"
	"elake-api-gateway/internal/service/concurrencyLimiter"
	"elake-api-gateway/internal/storage"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var globalApp *App

// New 创建一个新的应用实例
func New() *App {
	dbClient := repository.NewDBManager(storage.InitDB())
	redisClient := repository.NewRedisManager(storage.InitRedis())
	localCache := storage.NewLocalCache(time.Minute)
	ipdb := repository.NewIPDBManager(storage.InitIPDB())
	concurrencyLimiter.Init()
	limiter := concurrencyLimiter.Get()

	a := &App{
		DB:                 dbClient,
		Redis:              redisClient,
		LocalCache:         localCache,
		IPDB:               ipdb,
		ConcurrencyLimiter: limiter,
	}
	globalApp = a
	return a
}

// GetRedisClient 获取底层 Redis 客户端(用于 Pipeline 等高级操作)
func GetRedisClient() *redis.Client {
	if globalApp == nil || globalApp.Redis == nil {
		return nil
	}
	return globalApp.Redis.GetClient()
}

// LoadConfigFromDB 从数据库加载配置
func (app *App) LoadConfigFromDB() (*config.DatabaseConfig, error) {
	// 从数据库读取配置
	dbConfig, err := config.LoadDatabaseConfig(app.DB.GetDB())
	if err != nil {
		logger.Log.Error("从数据库加载配置失败", zap.Error(err))
		return nil, err
	}
	return dbConfig, nil
}

// Close 关闭应用实例的所有资源
func (app *App) Close() {
	// 关闭数据库连接
	if app.DB != nil {
		logger.Log.Info("关闭 DB 连接")
		err := app.DB.Close()
		if err != nil {
			logger.Log.Fatal("关闭 DB 连接失败", zap.Error(err))
		}
	}
	// 关闭 Redis 连接
	if app.Redis != nil {
		logger.Log.Info("关闭 Redis 连接")
		err := app.Redis.Close()
		if err != nil {
			logger.Log.Fatal("关闭 Redis 连接失败", zap.Error(err))
		}
	}
	// 关闭本地缓存
	if app.LocalCache != nil {
		logger.Log.Info("关闭 LocalCache")
		app.LocalCache.Stop()
	}
	// 关闭 IPDB 连接
	if app.IPDB != nil {
		logger.Log.Info("关闭 IPDB 连接")
		app.IPDB.Close()
	}
}
