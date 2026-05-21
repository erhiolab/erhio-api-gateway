package pubSub

// MessageData 消息数据
type MessageData struct {
	SecretID  string `json:"secretId,omitempty"`
	ServiceID int64  `json:"serviceId,omitempty"`
	RouteID   int64  `json:"routeId,omitempty"`
}

// Message 消息结构
type Message struct {
	Type       MessageType `json:"type"`
	Data       MessageData `json:"data"`
	GatewayIDs []string    `json:"gatewayIds,omitempty"`
}
