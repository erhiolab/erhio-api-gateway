package config

// DatabaseConfig 从数据库加载的配置
type DatabaseConfig struct {
	LocalCacheExpire int        `yaml:"local-cache-expire"`
	RedisCacheExpire int        `yaml:"redis-cache-expire"`
	ApiRoot          string     `yaml:"api-root"`
	NodeTimeout      int        `yaml:"node-timeout"`
	TotalTimeout     int        `yaml:"total-timeout"`
	Auth             AuthConfig `yaml:"auth"`
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
