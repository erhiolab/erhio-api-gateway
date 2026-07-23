package config

// DatabaseConfig 从数据库加载的配置
type DatabaseConfig struct {
	LocalCacheExpire int         `yaml:"local-cache-expire"`
	RedisCacheExpire int         `yaml:"redis-cache-expire"`
	ApiRoot          string      `yaml:"api-root"`
	NodeTimeout      int         `yaml:"node-timeout"`
	TotalTimeout     int         `yaml:"total-timeout"`
	Email            EmailConfig `yaml:"email"`
	Auth             AuthConfig  `yaml:"auth"`
}

// EmailConfig 邮件配置
type EmailConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Timeout  int    `yaml:"timeout"`
	MaxRetry int    `yaml:"max-retry"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	MasterKey       string `yaml:"master-key"`
	TimestampWindow int64  `yaml:"timestamp-window"`
	NonceWindow     int64  `yaml:"nonce-window"`
	NonceWindowHour int64  `yaml:"nonce-window-hour"`
	QpsLimit        int64  `yaml:"qps-limit"`
	QpmLimit        int64  `yaml:"qpm-limit"`
}
