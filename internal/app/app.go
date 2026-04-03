package app

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/repository"
	"elake-api-gateway/internal/storage"
	"time"

	"go.uber.org/zap"
)

// New 创建一个新的应用实例
func New() *App {
	// 初始化数据库
	dbClient := repository.NewDBManager(storage.InitDB())
	// 初始化 Redis
	redisClient := repository.NewRedisManager(storage.InitRedis())
	// 初始化本地缓存
	localCache := storage.NewLocalCache(time.Minute)
	// 初始化 IPDB
	ipdb := repository.NewIPDBManager(storage.InitIPDB())

	return &App{
		DB:         dbClient,
		Redis:      redisClient,
		LocalCache: localCache,
		IPDB:       ipdb,
	}
}

// LoadConfigFromDB 从数据库加载配置
func (app *App) LoadConfigFromDB() ([]config.Service, []config.Route, *config.DatabaseConfig, error) {
	// 从数据库读取所有服务
	services, err := app.DB.GetAllServices()
	if err != nil {
		logger.Log.Error("从数据库加载服务失败", zap.Error(err))
		return nil, nil, nil, err
	}
	// 从数据库读取所有路由
	routes, err := app.DB.GetAllRoutes()
	if err != nil {
		logger.Log.Error("从数据库加载路由失败", zap.Error(err))
		return nil, nil, nil, err
	}
	// 从数据库读取配置
	dbConfig, err := config.LoadDatabaseConfig(app.DB.GetDB())
	if err != nil {
		logger.Log.Error("从数据库加载配置失败", zap.Error(err))
		return nil, nil, nil, err
	}
	return services, routes, dbConfig, nil
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
