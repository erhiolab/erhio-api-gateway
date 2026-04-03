package config

// GatewayConfig 网关配置
type GatewayConfig struct {
	Port             int    `yaml:"port"`
	LocalCacheExpire int    `yaml:"local-cache-expire"`
	RedisCacheExpire int    `yaml:"redis-cache-expire"`
	DataPath         string `yaml:"data-path"`
	TempPath         string `yaml:"temp-path"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	MasterKey        string   `yaml:"master-key"`
	TimestampWindow  int64    `yaml:"timestamp-window"`
	NonceWindow      int64    `yaml:"nonce-window"`
	NonceWindowHour  int64    `yaml:"nonce-window-hour"`
	QpsLimit         int64    `yaml:"qps-limit"`
	QpmLimit         int64    `yaml:"qpm-limit"`
	IPBlacklist      []string `yaml:"ip-black-list"`
	CountryBlackList []string `yaml:"country-black-list"`
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
	DBHealthCheckInterval         int   `yaml:"db-health-check-interval"`
	DBHealthCheckFailThreshold    int32 `yaml:"db-health-check-fail-threshold"`
	DBHealthCheckOKThreshold      int32 `yaml:"db-health-check-ok-threshold"`
	RedisHealthCheckFailThreshold int32 `yaml:"redis-health-check-fail-threshold"`
	RedisHealthCheckOKThreshold   int32 `yaml:"redis-health-check-ok-threshold"`
	RedisHealthCheckInterval      int   `yaml:"redis-health-check-interval"`
	IPDBHealthCheckFailThreshold  int32 `yaml:"ipdb-health-check-fail-threshold"`
	IPDBHealthCheckOKThreshold    int32 `yaml:"ipdb-health-check-ok-threshold"`
	IPDBHealthCheckInterval       int   `yaml:"ipdb-health-check-interval"`
}

// DBConfig 数据库配置
type DBConfig struct {
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	User               string `yaml:"user"`
	Password           string `yaml:"password"`
	Name               string `yaml:"name"`
	MaxOpenConnections int    `yaml:"max-open-connections"`
	MaxIdleConnections int    `yaml:"max-idle-connections"`
	ConnMaxLifetime    int    `yaml:"conn-max-lifetime"`
	ConnMaxIdleTime    int    `yaml:"conn-max-idle-time"`
	ReadTimeout        int    `yaml:"read-timeout"`
	WriteTimeout       int    `yaml:"write-timeout"`
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
	ID           int64  `db:"id"`
	Path         string `db:"path"`
	Method       string `db:"method"`
	ServiceID    int64  `db:"service_id"`
	ServiceName  string `db:"service_name"`
	RequireAuth  bool   `db:"require_auth"`
	RequireLimit bool   `db:"require_limit"`
	QPS          int64  `db:"qps"`
	QPM          int64  `db:"qpm"`
	Enabled      bool   `db:"enabled"`
	IpLimit      bool   `db:"ip_limit"`
	CountryLimit bool   `db:"country_limit"`
}

// Service 服务配置
type Service struct {
	ID    int64  `db:"id"`
	Name  string `db:"name"`
	Nodes []ServiceNode
}

// ServiceNode 服务节点配置
type ServiceNode struct {
	ID        int64  `db:"id"`
	ServiceID int64  `db:"service_id"`
	NodeURL   string `db:"node_url"`
	Weight    int    `db:"weight"`
	Status    int    `db:"status"`
}

// IPDBConfig IPDB配置
type IPDBConfig struct {
	Token               string `yaml:"token"`
	MaxDownloadAttempts int    `yaml:"max-download-attempts"`
}

// Config 配置
type Config struct {
	Gateway  GatewayConfig `yaml:"gateway"`
	Auth     AuthConfig    `yaml:"auth"`
	Logger   LoggerConfig  `yaml:"logger"`
	Health   HealthConfig  `yaml:"health"`
	DB       DBConfig      `yaml:"db"`
	Redis    RedisConfig   `yaml:"redis"`
	IPDB     IPDBConfig    `yaml:"ipdb"`
	Routes   []Route       `yaml:"routes"`
	Services []Service     `yaml:"services"`
}
