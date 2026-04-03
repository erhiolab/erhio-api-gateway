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
	// UserAgentKey 客户端 User-Agent 键
	UserAgentKey contextKey = "user_agent"
	// ApiKeyInfoKey API密钥信息键值
	ApiKeyInfoKey contextKey = "api_key_info"
)
