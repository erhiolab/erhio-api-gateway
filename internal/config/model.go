package config

// GatewayConfig 网关配置
type GatewayConfig struct {
	Port int `yaml:"port"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Output         string `yaml:"output"`
	LogPath        string `yaml:"logPath"`
	RequestLogPath string `yaml:"requestLogPath"`
	MaxSize        int    `yaml:"maxSize"`
	MaxBackups     int    `yaml:"maxBackups"`
	MaxAge         int    `yaml:"maxAge"`
	Compress       bool   `yaml:"compress"`
}

// Route 路由配置
type Route struct {
	Path        string
	Method      string
	Service     string
	RequireAuth bool
	RateLimit   bool
}

// Service 服务配置
type Service struct {
	Name  string
	Nodes []string
}

// Config 配置
type Config struct {
	Gateway  GatewayConfig `yaml:"gateway"`
	Logger   LoggerConfig  `yaml:"logger"`
	Routes   []Route       `yaml:"routes"`
	Services []Service     `yaml:"services"`
}
