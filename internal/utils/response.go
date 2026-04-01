package utils

import (
	"encoding/json"
	"net/http"
	"time"
)

// JSONResponse 统一的JSON响应
func JSONResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		return
	}
}

// Error 错误响应
func Error(w http.ResponseWriter, status, errorCode int, message string) {
	JSONResponse(w, status, map[string]any{
		"code":      errorCode,
		"error":     message,
		"timestamp": time.Now().UnixMilli(),
	})
}

// BadRequest 错误的请求响应
func BadRequest(w http.ResponseWriter, name string) {
	Error(w, http.StatusBadRequest, 4000, "client "+name+" is empty")
}

// NotFound 路由不存在响应
func NotFound(w http.ResponseWriter) {
	Error(w, http.StatusNotFound, 4040, "route not found")
}

// TooManyRequests 请求过多
func TooManyRequests(w http.ResponseWriter) {
	Error(w, http.StatusTooManyRequests, 4290, "too many requests")
}

// InternalServerError 内部服务器错误
func InternalServerError(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, 5000, "internal server error")
}

// BadGateway 网关错误
func BadGateway(w http.ResponseWriter) {
	Error(w, http.StatusBadGateway, 5020, "bad gateway")
}
