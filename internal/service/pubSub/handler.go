package pubSub

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"
	"encoding/json"

	"go.uber.org/zap"
)

// MessageHandler 消息处理器
type MessageHandler struct {
	app *app.App
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(app *app.App) *MessageHandler {
	return &MessageHandler{app: app}
}

// HandleMessage 处理消息
func (h *MessageHandler) HandleMessage(message string) {
	var msg Message
	if err := json.Unmarshal([]byte(message), &msg); err != nil {
		logger.Log.Error("解析消息失败", zap.Error(err))
		return
	}
	switch msg.Type {
	case MessageTypeClearApiKeyCache:
		h.handleClearApiKeyCache(msg)
	case MessageTypeClearServiceCache:
		h.handleClearServiceCache(msg)
	case MessageTypeClearRouteCache:
		h.handleClearRouteCache(msg)
	default:
		logger.Log.Warn("消息处理器: 未知消息类型", zap.String("type", string(msg.Type)))
	}
}

// handleClearApiKeyCache 处理清除API密钥缓存消息
func (h *MessageHandler) handleClearApiKeyCache(msg Message) {
	var err error
	if msg.Data.SecretID == "" {
		err = h.app.ClearAllApiKeyInfoCache()
	} else {
		err = h.app.ClearApiKeyInfoCache(msg.Data.SecretID)
	}
	if err != nil {
		logger.Log.Error("消息处理器: 清除API密钥缓存失败",
			zap.String("secret_id", msg.Data.SecretID),
			zap.Error(err),
		)
	}
}

// handleClearServiceCache 处理清除服务缓存消息
func (h *MessageHandler) handleClearServiceCache(msg Message) {
	var err error
	if msg.Data.ServiceID == 0 {
		err = h.app.ClearAllServiceCache()
	} else {
		err = h.app.ClearServiceCache(msg.Data.ServiceID)
	}
	if err != nil {
		logger.Log.Error("消息处理器: 清除服务缓存失败",
			zap.Int64("service_id", msg.Data.ServiceID),
			zap.Error(err),
		)
	}
}

// handleClearRouteCache 处理清除路由缓存消息
func (h *MessageHandler) handleClearRouteCache(msg Message) {
	var err error
	if msg.Data.RouteID == 0 {
		err = h.app.ClearAllRouteCache()
	} else {
		err = h.app.ClearRouteCache(msg.Data.RouteID)
	}
	if err != nil {
		logger.Log.Error("消息处理器: 清除路由缓存失败",
			zap.Int64("route_id", msg.Data.RouteID),
			zap.Error(err),
		)
	}
}
