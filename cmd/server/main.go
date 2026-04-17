package main

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/gateway"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/middleware"
	"elake-api-gateway/internal/service/pubSub"
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

	// 启动消息订阅
	go func() {
		logger.Log.Info("消息处理器: 启动消息订阅", zap.String("channel", cfg.Redis.ProjectPrefix))
		pubSub.StartSubscription(appEngine, cfg.Redis.ProjectPrefix)
	}()

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
	proxy(appEngine)
	// 初始化API
	apiRoot := cfg.DatabaseConfig.ApiRoot
	if apiRoot == "" {
		apiRoot = "/_gateway/api"
	}
	api(appEngine, apiRoot)

	logger.Log.Info("监听端口: ", zap.String("port", strconv.Itoa(cfg.Gateway.Port)))
	logger.Log.Info("网关API根路由: ", zap.String("api-root", apiRoot))
	err = http.ListenAndServe(":"+strconv.Itoa(cfg.Gateway.Port), nil)
	if err != nil {
		logger.Log.Error("API网关启动失败", zap.Error(err))
		return
	}
}

// proxy 初始化代理
func proxy(app *app.App) {
	core := gateway.Handler(app)
	proxyHandler := middleware.Chain(
		core,
		middleware.Recovery(),
		middleware.TraceID(),
		middleware.HealthCheck(),
		middleware.ConcurrencyLimit(app),
		middleware.GetRealIP(app),
		middleware.Router(app),
		middleware.LoadBalancer(),
		middleware.GetUserAgent(),
		middleware.RateLimit(app),
		middleware.Logging(),
	)
	http.Handle("/", proxyHandler)
}

// api 初始化API
func api(app *app.App, apiRoot string) {
	apiHandler := gateway.APIHandler(app)
	apiMiddleware := middleware.Chain(
		apiHandler,
		middleware.Recovery(),
		middleware.TraceID(),
		middleware.HealthCheck(),
		middleware.ConcurrencyLimit(app),
		middleware.GetRealIP(app),
		middleware.GetUserAgent(),
		middleware.Logging(),
	)
	http.Handle(apiRoot+"/", http.StripPrefix(apiRoot, apiMiddleware))
}
