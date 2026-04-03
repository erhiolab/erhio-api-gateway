package models

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
