package controller

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/service/pubSub"
	"elake-api-gateway/internal/utils"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// PublishMessage 发布消息
func PublishMessage(app *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		p := utils.NewBodyParser(w, r)
		if !p.OK() {
			return
		}
		// 解析字段
		msgType := p.String("type")
		gatewayIDs := p.StringsOpt("gatewayIds")
		if !p.OK() {
			return
		}
		// 构建消息
		msg := pubSub.Message{
			Type:       pubSub.MessageType(msgType),
			GatewayIDs: gatewayIDs,
		}
		// 解析data字段
		if raw := p.Raw("data"); raw != nil {
			dataBytes, err := json.Marshal(raw)
			if err != nil {
				utils.BadRequest(w, "数据字段格式错误")
				return
			}
			var data pubSub.MessageData
			if err := json.Unmarshal(dataBytes, &data); err != nil {
				utils.BadRequest(w, "数据字段格式错误")
				return
			}
			msg.Data = data
		}
		// 发布消息
		cfg := config.Get()
		err := app.Redis.Publish(cfg.Redis.ProjectPrefix, msg)
		if err != nil {
			logger.WithRequestLogCtx(ctx, r).Error("发布消息失败",
				zap.String("type", msgType),
				zap.Error(err),
			)
			utils.InternalServerError(w, "发布消息失败")
			return
		}
		utils.Success(w, nil)
	}
}
