package main

import (
	"net/http"
	"strconv"

	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/gateway"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/middleware"

	"go.uber.org/zap"
)

// initRoutes 初始化代理路由
func initRoutes(app *app.App) {
	core := gateway.Handler(app)
	proxyHandler := middleware.Chain(
		core,
		middleware.Recovery(),
		middleware.TraceID(),
		middleware.HealthCheck(),
		middleware.ConcurrencyLimit(app),
		middleware.GetRealIP(app),
		middleware.GetUserAgent(),
		middleware.Router(app),
		middleware.LoadBalancer(),
		middleware.RateLimit(app),
		middleware.Logging(),
	)
	http.Handle("/", proxyHandler)
}

// initAPI 初始化API路由
func initAPI(app *app.App, cfg *config.Config) {
	apiRoot := cfg.DatabaseConfig.ApiRoot
	if apiRoot == "" {
		apiRoot = "/_gateway/api"
	}
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
	logger.Log.Info("网关API根路由: ", zap.String("api-root", apiRoot))
}

// startHTTPServer 启动HTTP服务
func startHTTPServer(cfg *config.Config) {
	logger.Log.Info("监听端口: ", zap.String("port", strconv.Itoa(cfg.Gateway.Port)))
	err := http.ListenAndServe(":"+strconv.Itoa(cfg.Gateway.Port), nil)
	if err != nil {
		logger.Log.Error("API网关启动失败", zap.Error(err))
		return
	}
}
