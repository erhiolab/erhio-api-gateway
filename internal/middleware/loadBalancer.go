package middleware

import (
	"context"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	// runtimeNodes 运行时节点映射
	runtimeNodes = make(map[int64][]*models.WeightedNode)
	// nodesMu 节点映射互斥锁
	nodesMu sync.RWMutex
)

// nodeFailureCooldown 节点失败冷却时间
const nodeFailureCooldown = 15 * time.Second

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
				utils.BadGateway(w)
				return
			}
			selectedNode := SelectNode(service, nil)
			if selectedNode == nil {
				logger.WithRequestLogCtx(ctx).Warn("负载均衡插件: 当前没有可用节点",
					zap.Int64("service_id", service.ID),
					zap.String("service_name", service.Name),
				)
				utils.BadGateway(w)
				return
			}
			ctx = context.WithValue(ctx, utils.SelectedNodeKey, &models.SelectedNode{Node: selectedNode})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SelectNode 选择一个当前可用的节点
func SelectNode(service *models.Service, excluded map[int64]struct{}) *models.ServiceNode {
	nodes := getRuntimeNodes(service)
	if len(nodes) == 0 {
		return nil
	}
	if len(excluded) > 0 {
		filtered := make([]*models.WeightedNode, 0, len(nodes))
		for _, node := range nodes {
			if _, skip := excluded[node.Node.ID]; skip {
				continue
			}
			filtered = append(filtered, node)
		}
		nodes = filtered
	}
	if len(nodes) == 0 {
		return nil
	}
	return selectSmoothNode(nodes)
}

// selectSmoothNode 平滑加权
func selectSmoothNode(nodes []*models.WeightedNode) *models.ServiceNode {
	if len(nodes) == 1 {
		return nodes[0].Node
	}
	var best *models.WeightedNode
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

// MarkNodeFailure 将节点标记为运行时暂时不可用
func MarkNodeFailure(serviceID, nodeID int64, err error) {
	nodesMu.Lock()
	defer nodesMu.Unlock()
	for _, node := range runtimeNodes[serviceID] {
		if node.Node.ID != nodeID {
			continue
		}
		node.ConsecutiveFails++
		node.DisabledUntil = time.Now().Add(nodeFailureCooldown)
		node.CurrentWeight = 0
		if err != nil {
			node.LastError = err.Error()
		}
		return
	}
}

// MarkNodeSuccess 节点请求成功后清理临时故障状态
func MarkNodeSuccess(serviceID, nodeID int64) {
	nodesMu.Lock()
	defer nodesMu.Unlock()
	for _, node := range runtimeNodes[serviceID] {
		if node.Node.ID != nodeID {
			continue
		}
		node.ConsecutiveFails = 0
		node.DisabledUntil = time.Time{}
		node.LastError = ""
		return
	}
}

// getRuntimeNodes 获取运行时的节点包装对象
func getRuntimeNodes(service *models.Service) []*models.WeightedNode {
	nodesMu.RLock()
	wnodes, exists := runtimeNodes[service.ID]
	nodesMu.RUnlock()
	if exists && runtimeNodesMatch(wnodes, service.Nodes) {
		return filterAvailableNodes(wnodes)
	}
	nodesMu.Lock()
	defer nodesMu.Unlock()
	current := runtimeNodes[service.ID]
	if runtimeNodesMatch(current, service.Nodes) {
		return filterAvailableNodes(current)
	}
	newNodes := rebuildRuntimeNodes(service, current)
	runtimeNodes[service.ID] = newNodes
	return filterAvailableNodes(newNodes)
}

// runtimeNodesMatch 检查当前运行时节点是否与配置节点匹配
func runtimeNodesMatch(current []*models.WeightedNode, nodes []models.ServiceNode) bool {
	if len(current) != len(nodes) {
		return false
	}
	for i := range nodes {
		if current[i].Node.ID != nodes[i].ID ||
			current[i].Node.NodeURL != nodes[i].NodeURL ||
			current[i].Node.Weight != nodes[i].Weight ||
			current[i].Node.Status != nodes[i].Status {
			return false
		}
	}
	return true
}

// rebuildRuntimeNodes 重建运行时节点映射
func rebuildRuntimeNodes(service *models.Service, current []*models.WeightedNode) []*models.WeightedNode {
	previous := make(map[int64]*models.WeightedNode, len(current))
	for _, node := range current {
		previous[node.Node.ID] = node
	}
	newNodes := make([]*models.WeightedNode, 0, len(service.Nodes))
	for i := range service.Nodes {
		serviceNode := service.Nodes[i]
		weightedNode := &models.WeightedNode{
			Node:          &serviceNode,
			CurrentWeight: 0,
		}
		if old, ok := previous[serviceNode.ID]; ok {
			weightedNode.CurrentWeight = old.CurrentWeight
			weightedNode.DisabledUntil = old.DisabledUntil
			weightedNode.ConsecutiveFails = old.ConsecutiveFails
			weightedNode.LastError = old.LastError
		}
		newNodes = append(newNodes, weightedNode)
	}
	return newNodes
}

// filterAvailableNodes 过滤出当前可用的节点
func filterAvailableNodes(nodes []*models.WeightedNode) []*models.WeightedNode {
	now := time.Now()
	active := make([]*models.WeightedNode, 0, len(nodes))
	for _, node := range nodes {
		if node.Node.Status != 1 {
			continue
		}
		if !node.DisabledUntil.IsZero() && node.DisabledUntil.After(now) {
			continue
		}
		active = append(active, node)
	}
	return active
}
