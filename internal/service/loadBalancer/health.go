package loadBalancer

import (
	"elake-api-gateway/internal/models"
	"time"
)

// MarkNodeFailure 将节点标记为运行时暂时不可用, 同时降低5%可用率
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
		// 直接降低可用率
		node.Node.Availability -= 5.0
		if node.Node.Availability < 0 {
			node.Node.Availability = 0
		}
		updateNodeAvailabilityToDB(node.Node.ID, node.Node.Availability)
		return
	}
}

// MarkNodeSuccess 节点请求成功后清理临时故障状态, 同时恢复0.5%可用率
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
		// 成功时缓慢恢复可用率, 但不超过100
		if node.Node.Availability < 100 {
			node.Node.Availability += 0.5
			if node.Node.Availability > 100 {
				node.Node.Availability = 100
			}
			updateNodeAvailabilityToDB(node.Node.ID, node.Node.Availability)
		}
		return
	}
}

// MarkNodeMaxConn 节点最大连接数超过限制, 同时降低3%可用率
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
		// 降低可用率
		node.Node.Availability -= 3.0
		if node.Node.Availability < 0 {
			node.Node.Availability = 0
		}
		updateNodeAvailabilityToDB(node.Node.ID, node.Node.Availability)
		return
	}
}
