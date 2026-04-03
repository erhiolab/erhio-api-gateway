package models

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
