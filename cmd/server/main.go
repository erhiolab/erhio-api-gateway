package main

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/gateway"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/middleware"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

// main 主函数
func main() {
	// 初始化配置
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

	// 创建应用实例
	appEngine := app.New()
	defer appEngine.Close()

	// 初始化路由
	core := gateway.Handler(appEngine)
	handler := middleware.Chain(
		core,
		middleware.Recovery(),
		middleware.Router(),
		middleware.RequestID(),
		middleware.RealIP(),
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
