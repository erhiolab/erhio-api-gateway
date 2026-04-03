package config

// DatabaseConfig 从数据库加载的配置
type DatabaseConfig struct {
	LocalCacheExpire int        `yaml:"local-cache-expire"`
	RedisCacheExpire int        `yaml:"redis-cache-expire"`
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
	ID       int64  `db:"id"`
	Name     string `db:"name"`
	BasePath string `db:"base_path"`
	Nodes    []ServiceNode
}

// ServiceNode 服务节点配置
type ServiceNode struct {
	ID        int64  `db:"id"`
	ServiceID int64  `db:"service_id"`
	NodeURL   string `db:"node_url"`
	Weight    int    `db:"weight"`
	Status    int    `db:"status"`
}
