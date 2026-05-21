package pubSub

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"encoding/json"
	"slices"

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

// shouldProcessMessage 判断当前网关是否应该处理该消息
func (h *MessageHandler) shouldProcessMessage(gatewayIDs []string) bool {
	if len(gatewayIDs) == 0 {
		return true
	}
	currentGatewayID := config.Get().Gateway.ID
	return slices.Contains(gatewayIDs, currentGatewayID)
}

// HandleMessage 处理消息
func (h *MessageHandler) HandleMessage(message string) {
	var msg Message
	if err := json.Unmarshal([]byte(message), &msg); err != nil {
		logger.Log.Error("解析消息失败", zap.Error(err))
		return
	}
	if !h.shouldProcessMessage(msg.GatewayIDs) {
		logger.Log.Debug("消息处理器: 当前网关不在目标列表中, 跳过处理",
			zap.String("gateway_id", config.Get().Gateway.ID),
			zap.Strings("target_gateway_ids", msg.GatewayIDs),
		)
		return
	}
	switch msg.Type {
	case MessageTypeReloadConfig:
		h.handleReloadConfig()
	case MessageTypeTriggerIPDBUpdate:
		h.handleTriggerIPDBUpdate()
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

// handleReloadConfig 处理重载数据库配置消息
func (h *MessageHandler) handleReloadConfig() {
	logger.Log.Info("消息处理器: 开始重载数据库配置", zap.Any("current_config", config.Get()))
	// 从数据库加载配置
	dbConfig, err := h.app.LoadConfigFromDB()
	if err != nil {
		logger.Log.Error("消息处理器: 从数据库加载配置失败", zap.Error(err))
		return
	}
	// 合并配置到内存
	cfg := config.Get()
	mergedCfg := config.MergeConfig(cfg, dbConfig)
	config.Set(mergedCfg)
	logger.Log.Info("消息处理器: 重载数据库配置成功", zap.Any("new_config", mergedCfg))
}

// handleTriggerIPDBUpdate 处理触发IPDB更新消息
func (h *MessageHandler) handleTriggerIPDBUpdate() {
	err := h.app.IPDB.TriggerIPDBUpdate()
	if err != nil {
		logger.Log.Error("消息处理器: 触发IPDB更新失败", zap.Error(err))
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
