package main

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/gateway"
	"elake-api-gateway/internal/middleware"
	"net/http"
)

// main 主函数
func main() {
	cfg := config.Load()
	config.Set(cfg)
	core := http.HandlerFunc(gateway.CoreHandler)
	handler := middleware.Chain(
		core,
		middleware.Logging(),
	)
	http.Handle("/", handler)
	http.ListenAndServe(":8080", nil)
}
