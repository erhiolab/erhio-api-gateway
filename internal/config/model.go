package config

// Route 路由配置
type Route struct {
	Path        string
	Method      string
	Service     string
	RequireAuth bool
}

// Service 服务配置
type Service struct {
	Name  string
	Nodes []string
}

// Config 配置
type Config struct {
	Routes   []Route
	Services []Service
}
