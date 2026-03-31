package config

// GatewayConfig 网关配置
type GatewayConfig struct {
	Port     int   `yaml:"port"`
	QpsLimit int64 `yaml:"qps-limit"`
	QpmLimit int64 `yaml:"qpm-limit"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Output         string `yaml:"output"`
	LogPath        string `yaml:"log-path"`
	RequestLogPath string `yaml:"request-log-path"`
	MaxSize        int    `yaml:"max-size"`
	MaxBackups     int    `yaml:"max-backups"`
	MaxAge         int    `yaml:"max-age"`
	Compress       bool   `yaml:"compress"`
}

// HealthConfig 健康检查配置
type HealthConfig struct {
	RedisHealthCheckFailThreshold int32 `yaml:"redis-health-check-fail-threshold"`
	RedisHealthCheckOKThreshold   int32 `yaml:"redis-health-check-ok-threshold"`
	RedisHealthCheckInterval      int   `yaml:"redis-health-check-interval"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host                  string `yaml:"host"`
	Port                  int    `yaml:"port"`
	Password              string `yaml:"password"`
	DB                    int    `yaml:"db"`
	PoolSize              int    `yaml:"pool-size"`
	ProjectPrefix         string `yaml:"project-prefix"`
	MinIdleConnections    int    `yaml:"min-idle-connections"`
	ConnectionMaxIdleTime int    `yaml:"connection-max-idle-time"`
	DialTimeout           int    `yaml:"dial-timeout"`
	ReadTimeout           int    `yaml:"read-timeout"`
	WriteTimeout          int    `yaml:"write-timeout"`
}

// Route 路由配置
type Route struct {
	Path         string `yaml:"path"`
	Method       string `yaml:"method"`
	Service      string `yaml:"service"`
	RequireAuth  bool   `yaml:"require-auth"`
	RequireLimit bool   `yaml:"require-limit"`
}

// Service 服务配置
type Service struct {
	Name  string   `yaml:"name"`
	Nodes []string `yaml:"nodes"`
}

// Config 配置
type Config struct {
	Gateway  GatewayConfig `yaml:"gateway"`
	Logger   LoggerConfig  `yaml:"logger"`
	Health   HealthConfig  `yaml:"health"`
	Redis    RedisConfig   `yaml:"redis"`
	Routes   []Route       `yaml:"routes"`
	Services []Service     `yaml:"services"`
}
