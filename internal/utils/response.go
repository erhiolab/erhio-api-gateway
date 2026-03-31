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
