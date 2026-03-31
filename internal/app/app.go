package app

import (
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/repository"
	"elake-api-gateway/internal/storage"

	"go.uber.org/zap"
)

// New 创建一个新的应用实例
func New() *App {
	// 初始化 Redis
	client := storage.InitRedis()

	return &App{
		Redis: repository.NewRedisManager(client),
	}
}

// Close 关闭应用实例的所有资源
func (app *App) Close() {
	if app.Redis != nil {
		logger.Log.Info("关闭Redis")
		err := app.Redis.Close()
		if err != nil {
			logger.Log.Fatal("关闭 Redis 连接失败", zap.Error(err))
		}
	}
}
