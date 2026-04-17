package pubSub

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// StartSubscription 启动订阅
func StartSubscription(app *app.App, channel string) {
	pubSub := app.Redis.Subscribe(channel)
	defer func(pubSub *redis.PubSub) {
		_ = pubSub.Close()
	}(pubSub)
	handler := NewMessageHandler(app)
	ch := pubSub.Channel()
	for msg := range ch {
		logger.Log.Info("收到消息", zap.String("channel", msg.Channel), zap.String("message", msg.Payload))
		handler.HandleMessage(msg.Payload)
	}
}
