package app

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/repository"
	"elake-api-gateway/internal/storage"

	"go.uber.org/zap"
)

// New 创建一个新的应用实例
func New() *App {
	// 初始化数据库
	dbClient := storage.InitDB()
	// 初始化 Redis
	redisClient := storage.InitRedis()

	return &App{
		DB:    repository.NewDBManager(dbClient),
		Redis: repository.NewRedisManager(redisClient),
	}
}

// LoadConfigFromDB 从数据库加载配置
func (app *App) LoadConfigFromDB() ([]config.Service, []config.Route, error) {
	// 从数据库读取所有服务
	services, err := app.DB.GetAllServices()
	if err != nil {
		logger.Log.Error("从数据库加载服务失败", zap.Error(err))
		return nil, nil, err
	}
	// 从数据库读取所有路由
	routes, err := app.DB.GetAllRoutes()
	if err != nil {
		logger.Log.Error("从数据库加载路由失败", zap.Error(err))
		return nil, nil, err
	}
	return services, routes, nil
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
}
