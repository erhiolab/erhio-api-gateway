package api

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

// publishMessageRequest 发布消息请求
type publishMessageRequest struct {
	Type       string          `json:"type"`
	Data       json.RawMessage `json:"data"`
	GatewayIDs []string        `json:"gatewayIds"`
}

// PublishMessage 发布消息
func PublishMessage(app *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// 解析请求体
		var req publishMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.BadRequest(w, "请求体格式错误")
			return
		}
		// 验证type字段
		if req.Type == "" {
			utils.BadRequest(w, "消息类型不能为空")
			return
		}
		// 构建消息
		msg := pubSub.Message{
			Type:       pubSub.MessageType(req.Type),
			GatewayIDs: req.GatewayIDs,
		}
		// 解析data字段
		var data pubSub.MessageData
		if len(req.Data) > 0 {
			if err := json.Unmarshal(req.Data, &data); err != nil {
				utils.BadRequest(w, "数据字段格式错误")
				return
			}
			msg.Data = data
		}
		// 发布消息
		cfg := config.Get()
		err := app.Redis.Publish(cfg.Redis.ProjectPrefix, msg)
		if err != nil {
			logger.WithRequestLogCtx(ctx).Error("发布消息失败",
				zap.String("type", req.Type),
				zap.Error(err),
			)
			utils.InternalServerError(w, "发布消息失败")
			return
		}
		utils.Success(w, nil)
	}
}
