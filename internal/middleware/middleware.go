package middleware

import "net/http"

// Middleware 中间件
type Middleware func(http.Handler) http.Handler

// ResponseWriter 自定义响应写入器
type ResponseWriter struct {
	http.ResponseWriter
	StatusCode  int
	Size        int
	WroteHeader bool
}
