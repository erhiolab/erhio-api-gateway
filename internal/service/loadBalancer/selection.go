package loadBalancer

import (
	"elake-api-gateway/internal/models"
	"time"
)

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
		// 考虑可用度百分比调整权重
		adjustedWeight := max(int(float64(node.Node.Weight)*(node.Node.Availability/100)), 1)
		node.CurrentWeight += adjustedWeight
		totalWeight += adjustedWeight
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

// normalizedWeight 归一化节点权重
func normalizedWeight(weight int) int {
	if weight > 0 {
		return weight
	}
	return 1
}
