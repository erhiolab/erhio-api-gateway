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

// WriteHeader 写入状态码
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.StatusCode = code
	rw.WroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

// Write 写入响应
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	if !rw.WroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.Size += n
	return n, err
}
