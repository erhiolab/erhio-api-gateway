package middleware

import (
	"context"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/service/loadBalancer"
	"elake-api-gateway/internal/utils"
	"net/http"

	"go.uber.org/zap"
)

// LoadBalancer 负载均衡插件
func LoadBalancer() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			service, ok := ctx.Value(utils.ServiceKey).(*models.Service)
			if !ok || service == nil || len(service.Nodes) == 0 {
				logger.WithRequestLogCtx(ctx).Warn("负载均衡插件: 服务没有活动节点",
					zap.Int64("service_id", service.ID),
					zap.String("service_name", service.Name),
				)
				utils.BadGateway(w, "服务没有活动节点")
				return
			}
			selectedNode := loadBalancer.SelectNode(service, nil)
			if selectedNode == nil {
				logger.WithRequestLogCtx(ctx).Warn("负载均衡插件: 当前没有可用节点",
					zap.Int64("service_id", service.ID),
					zap.String("service_name", service.Name),
				)
				utils.BadGateway(w, "当前没有可用节点")
				return
			}
			ctx = context.WithValue(ctx, utils.SelectedNodeKey, &models.SelectedNode{Node: selectedNode})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
