package main

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/service/loadBalancer"
	"elake-api-gateway/internal/service/pubSub"
	"elake-api-gateway/internal/utils"

	"go.uber.org/zap"
)

// main 主函数
func main() {
	// 加载基础配置
	cfg := loadBaseConfig()
	if cfg == nil {
		return
	}

	// 初始化日志
	logger.InitLogger()
	defer func(Log *zap.Logger) {
		_ = Log.Sync()
	}(logger.Log)

	// 创建目录
	utils.CreateFolder()

	// 创建应用实例
	appEngine := createAppEngine()
	defer appEngine.Close()

	// 初始化负载均衡器
	loadBalancer.Init(appEngine.DB)

	// 启动消息订阅
	startSubscription(appEngine, cfg)

	// 加载数据库配置
	mergedCfg := loadDatabaseConfig(cfg, appEngine)
	if mergedCfg == nil {
		return
	}

	// 初始化路由和API
	initRoutes(appEngine)
	initAPI(appEngine, mergedCfg)

	// 启动HTTP服务
	startHTTPServer(mergedCfg)
}

// loadBaseConfig 加载基础配置
func loadBaseConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}
	config.Set(cfg)
	return cfg
}

// createAppEngine 创建应用实例
func createAppEngine() *app.App {
	return app.New()
}

// startSubscription 启动消息订阅
func startSubscription(appEngine *app.App, cfg *config.Config) {
	go func() {
		logger.Log.Info("消息处理器: 启动消息订阅", zap.String("channel", cfg.Redis.ProjectPrefix))
		pubSub.StartSubscription(appEngine, cfg.Redis.ProjectPrefix)
	}()
}

// loadDatabaseConfig 加载数据库配置
func loadDatabaseConfig(cfg *config.Config, appEngine *app.App) *config.Config {
	dbConfig, err := appEngine.LoadConfigFromDB()
	if err != nil {
		logger.Log.Fatal("从数据库加载配置失败", zap.Error(err))
		return nil
	}
	mergedCfg := config.MergeConfig(cfg, dbConfig)
	config.Set(mergedCfg)
	return mergedCfg
}
