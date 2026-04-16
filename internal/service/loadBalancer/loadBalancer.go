package loadBalancer

import (
	"elake-api-gateway/internal/models"
	"sync"
	"time"
)

var (
	runtimeNodes = make(map[int64][]*models.WeightedNode)
	nodesMu      sync.RWMutex
)

// nodeFailureCooldown 节点故障冷却时间
const nodeFailureCooldown = 15 * time.Second

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
	return selectNodeByLoad(nodes)
}

// MarkNodeFailure 将节点标记为运行时暂时不可用
func MarkNodeFailure(serviceID, nodeID int64, err error) {
	nodesMu.Lock()
	defer nodesMu.Unlock()
	for _, node := range runtimeNodes[serviceID] {
		if node.Node.ID != nodeID {
			continue
		}
		if node.ActiveRequests > 0 {
			node.ActiveRequests--
		}
		node.ConsecutiveFails++
		node.DisabledUntil = time.Now().Add(nodeFailureCooldown)
		node.CurrentWeight = 0
		node.CircuitBreaker = &models.CircuitBreaker{
			Type:      models.CircuitBreakerError,
			StartTime: time.Now(),
			Reason:    "consecutive errors",
		}
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
		if node.ActiveRequests > 0 {
			node.ActiveRequests--
		}
		node.ConsecutiveFails = 0
		node.DisabledUntil = time.Time{}
		node.LastError = ""
		node.CircuitBreaker = nil
		return
	}
}

// MarkNodeMaxConn 节点最大连接数超过限制
func MarkNodeMaxConn(serviceID, nodeID int64) {
	nodesMu.Lock()
	defer nodesMu.Unlock()
	for _, node := range runtimeNodes[serviceID] {
		if node.Node.ID != nodeID {
			continue
		}
		node.CircuitBreaker = &models.CircuitBreaker{
			Type:      models.CircuitBreakerMaxConn,
			StartTime: time.Now(),
			Reason:    "max connections exceeded",
		}
		return
	}
}

// ReleaseNodeRequest 释放节点请求
func ReleaseNodeRequest(serviceID, nodeID int64) {
	nodesMu.Lock()
	defer nodesMu.Unlock()
	for _, node := range runtimeNodes[serviceID] {
		if node.Node.ID != nodeID {
			continue
		}
		if node.ActiveRequests > 0 {
			node.ActiveRequests--
		}
		return
	}
}

// selectNodeByLoad 选择一个当前可用的节点
func selectNodeByLoad(nodes []*models.WeightedNode) *models.ServiceNode {
	nodesMu.Lock()
	defer nodesMu.Unlock()
	available := filterAvailableNodesWithLock(nodes)
	if len(available) == 0 {
		return nil
	}
	if len(available) == 1 {
		available[0].ActiveRequests++
		return available[0].Node
	}
	candidates := make([]*models.WeightedNode, 0, len(available))
	for _, node := range available {
		if len(candidates) == 0 {
			candidates = append(candidates, node)
			continue
		}
		best := candidates[0]
		switch compareNodeLoad(node, best) {
		case -1:
			candidates = []*models.WeightedNode{node}
		case 0:
			candidates = append(candidates, node)
		}
	}
	var best *models.WeightedNode
	totalWeight := 0
	for _, node := range candidates {
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
	best.ActiveRequests++
	return best.Node
}

// filterAvailableNodesWithLock 过滤出当前可用的节点
func filterAvailableNodesWithLock(nodes []*models.WeightedNode) []*models.WeightedNode {
	now := time.Now()
	active := make([]*models.WeightedNode, 0, len(nodes))
	for _, node := range nodes {
		if node.Node.Status != 1 {
			continue
		}
		if cb := node.CircuitBreaker; cb != nil && cb.IsActive() {
			continue
		}
		if !node.DisabledUntil.IsZero() && node.DisabledUntil.After(now) {
			continue
		}
		maxConn := node.Node.MaxConn
		if maxConn <= 0 {
			maxConn = 100
		}
		if node.ActiveRequests >= maxConn {
			continue
		}
		active = append(active, node)
	}
	return active
}

// compareNodeLoad 比较两个节点的负载
func compareNodeLoad(a, b *models.WeightedNode) int {
	left := a.ActiveRequests * normalizedWeight(b.Node.Weight)
	right := b.ActiveRequests * normalizedWeight(a.Node.Weight)
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

// getRuntimeNodes 获取当前运行时节点
func getRuntimeNodes(service *models.Service) []*models.WeightedNode {
	nodesMu.RLock()
	wnodes, exists := runtimeNodes[service.ID]
	nodesMu.RUnlock()
	if exists && runtimeNodesMatch(wnodes, service.Nodes) {
		return wnodes
	}
	nodesMu.Lock()
	defer nodesMu.Unlock()
	current := runtimeNodes[service.ID]
	if runtimeNodesMatch(current, service.Nodes) {
		return current
	}
	newNodes := rebuildRuntimeNodes(service, current)
	runtimeNodes[service.ID] = newNodes
	return newNodes
}

// runtimeNodesMatch 检查当前运行时节点是否匹配配置
func runtimeNodesMatch(current []*models.WeightedNode, nodes []models.ServiceNode) bool {
	if len(current) != len(nodes) {
		return false
	}
	for i := range nodes {
		if current[i].Node.ID != nodes[i].ID ||
			current[i].Node.NodeURL != nodes[i].NodeURL ||
			current[i].Node.Weight != nodes[i].Weight ||
			current[i].Node.MaxConn != nodes[i].MaxConn ||
			current[i].Node.Status != nodes[i].Status {
			return false
		}
	}
	return true
}

// rebuildRuntimeNodes 重建构建当前运行时节点
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
			weightedNode.ActiveRequests = old.ActiveRequests
			weightedNode.DisabledUntil = old.DisabledUntil
			weightedNode.ConsecutiveFails = old.ConsecutiveFails
			weightedNode.LastError = old.LastError
			weightedNode.CircuitBreaker = old.CircuitBreaker
		}
		newNodes = append(newNodes, weightedNode)
	}
	return newNodes
}

// normalizedWeight 归一化节点权重
func normalizedWeight(weight int) int {
	if weight > 0 {
		return weight
	}
	return 1
}
