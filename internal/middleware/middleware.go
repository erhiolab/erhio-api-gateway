package middleware

import "net/http"

// Middleware 中间件
type Middleware func(http.Handler) http.Handler
