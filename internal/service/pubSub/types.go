package pubSub

// MessageType 消息类型
type MessageType string

const (
	// MessageTypeReloadConfig 重载数据库配置
	MessageTypeReloadConfig MessageType = "reload_config"
	// MessageTypeTriggerIPDBUpdate 触发IPDB自动更新
	MessageTypeTriggerIPDBUpdate MessageType = "trigger_ipdb_update"
	// MessageTypeClearApiKeyCache 清除API密钥缓存
	MessageTypeClearApiKeyCache MessageType = "clear_api_key_cache"
	// MessageTypeClearServiceCache 清除服务缓存
	MessageTypeClearServiceCache MessageType = "clear_service_cache"
	// MessageTypeClearRouteCache 清除路由缓存
	MessageTypeClearRouteCache MessageType = "clear_route_cache"
)
