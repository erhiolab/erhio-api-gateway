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

// Success 成功响应
func Success(w http.ResponseWriter, data any) {
	JSONResponse(w, http.StatusOK, map[string]any{
		"code":      2000,
		"data":      data,
		"timestamp": time.Now().UnixMilli(),
	})
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
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, 4000, message)
}

// Unauthorized 未授权响应
func Unauthorized(w http.ResponseWriter, message ...string) {
	messageStr := "unauthorized"
	if len(message) > 0 {
		messageStr = message[0]
	}
	Error(w, http.StatusUnauthorized, 4010, messageStr)
}

// Forbidden 禁止响应
func Forbidden(w http.ResponseWriter, message ...string) {
	messageStr := "forbidden"
	if len(message) > 0 {
		messageStr = message[0]
	}
	Error(w, http.StatusForbidden, 4030, messageStr)
}

// NotFound 路由不存在响应
func NotFound(w http.ResponseWriter) {
	Error(w, http.StatusNotFound, 4040, "route not found")
}

// Conflict 冲突响应
func Conflict(w http.ResponseWriter) {
	Error(w, http.StatusConflict, 4090, "conflict")
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

// ServerBusy 服务器匆忙
func ServerBusy(w http.ResponseWriter) {
	Error(w, http.StatusServiceUnavailable, 5030, "server busy")
}
