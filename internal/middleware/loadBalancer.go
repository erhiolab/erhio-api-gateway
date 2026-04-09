package middleware

import (
	"context"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"sync"

	"go.uber.org/zap"
)

// WeightedNode 用于平滑加权轮询的运行时节点
type WeightedNode struct {
	Node          *models.ServiceNode
	CurrentWeight int
	mu            sync.Mutex
}

var (
	// runtimeNodes 运行时节点映射
	runtimeNodes = make(map[int64][]*WeightedNode)
	// nodesMu 节点映射互斥锁
	nodesMu sync.RWMutex
)

// LoadBalancer 负载均衡插件
func LoadBalancer() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			service, ok := ctx.Value(utils.ServiceKey).(*models.Service)
			if !ok || len(service.Nodes) == 0 {
				logger.WithRequestLogCtx(ctx).Warn("负载均衡插件: 服务没有活动节点, 无法负载均衡",
					zap.Int64("service_id", service.ID),
					zap.String("service_name", service.Name),
				)
				utils.BadGateway(w)
				return
			}
			nodes := getRuntimeNodes(service)
			if len(nodes) == 0 {
				logger.WithRequestLogCtx(ctx).Warn("负载均衡插件: 服务没有活动节点, 无法负载均衡",
					zap.Int64("service_id", service.ID),
					zap.String("service_name", service.Name),
				)
				utils.BadGateway(w)
				return
			}
			selected := selectSmoothNode(nodes)
			ctx = context.WithValue(ctx, utils.SelectedNodeKey, selected)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// selectSmoothNode 平滑加权
func selectSmoothNode(nodes []*WeightedNode) *models.ServiceNode {
	if len(nodes) == 1 {
		return nodes[0].Node
	}
	var best *WeightedNode
	totalWeight := 0
	nodesMu.Lock()
	defer nodesMu.Unlock()
	for _, node := range nodes {
		node.CurrentWeight += node.Node.Weight
		totalWeight += node.Node.Weight
		if best == nil || node.CurrentWeight > best.CurrentWeight {
			best = node
		}
	}
	if best == nil {
		return nil
	}
	best.CurrentWeight -= totalWeight
	return best.Node
}

// getRuntimeNodes 获取运行时的节点包装对象
func getRuntimeNodes(service *models.Service) []*WeightedNode {
	nodesMu.RLock()
	wnodes, exists := runtimeNodes[service.ID]
	nodesMu.RUnlock()
	if exists && len(wnodes) == len(service.Nodes) {
		needUpdate := false
		for i := range service.Nodes {
			if wnodes[i].Node.NodeURL != service.Nodes[i].NodeURL ||
				wnodes[i].Node.Weight != service.Nodes[i].Weight ||
				wnodes[i].Node.Status != service.Nodes[i].Status {
				needUpdate = true
				break
			}
		}
		if !needUpdate {
			var active []*WeightedNode
			for _, n := range wnodes {
				if n.Node.Status == 1 {
					active = append(active, n)
				}
			}
			return active
		}
	}
	// 初始化运行时节点
	nodesMu.Lock()
	defer nodesMu.Unlock()
	var newNodes []*WeightedNode
	for i := range service.Nodes {
		newNodes = append(newNodes, &WeightedNode{
			Node:          &service.Nodes[i],
			CurrentWeight: 0,
		})
	}
	runtimeNodes[service.ID] = newNodes
	return newNodes
}
