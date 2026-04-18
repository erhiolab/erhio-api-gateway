package loadBalancer

import "elake-api-gateway/internal/models"

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
