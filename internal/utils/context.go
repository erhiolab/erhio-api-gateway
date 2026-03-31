package utils

// contextKey 上下文键
type contextKey string

const (
	// RouteKey 路由键
	RouteKey contextKey = "route"
	// ServiceKey 服务键
	ServiceKey contextKey = "service"
	// RequestIDKey 请求ID键
	RequestIDKey contextKey = "request_id"
	// ClientIPKey 客户端IP键
	ClientIPKey contextKey = "client_ip"
	// SecretID API 密钥ID
	SecretID contextKey = "secret_id"
)
