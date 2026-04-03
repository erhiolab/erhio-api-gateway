package main

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/gateway"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/middleware"
	"elake-api-gateway/internal/utils"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

// main 主函数
func main() {
	// 加载基础配置
	cfg, err := config.Load()
	if err != nil {
		return
	}
	config.Set(cfg)

	// 初始化日志
	logger.InitLogger()
	defer func(Log *zap.Logger) {
		_ = Log.Sync()
	}(logger.Log)

	// 创建目录
	utils.CreateFolder()

	// 创建应用实例
	appEngine := app.New()
	defer appEngine.Close()

	// 从数据库加载配置
	dbConfig, err := appEngine.LoadConfigFromDB()
	if err != nil {
		logger.Log.Fatal("从数据库加载配置失败", zap.Error(err))
		return
	}
	// 合并配置到内存
	mergedCfg := config.MergeConfig(cfg, dbConfig)
	config.Set(mergedCfg)
	// 初始化路由
	core := gateway.Handler(appEngine)
	handler := middleware.Chain(
		core,
		middleware.Recovery(),
		middleware.HealthCheck(),
		middleware.Router(appEngine),
		middleware.GetUserAgent(),
		middleware.GetRealIP(appEngine),
		middleware.TraceID(),
		middleware.Logging(),
	)

	logger.Log.Info("监听端口: ", zap.String("port", strconv.Itoa(cfg.Gateway.Port)))
	http.Handle("/", handler)
	err = http.ListenAndServe(":"+strconv.Itoa(cfg.Gateway.Port), nil)
	if err != nil {
		logger.Log.Error("API网关启动失败", zap.Error(err))
		return
	}
}
