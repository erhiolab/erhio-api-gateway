package models

import (
	"sync"
	"time"
)

// WeightedNode 平滑权重节点
type WeightedNode struct {
	Node             *ServiceNode
	CurrentWeight    int
	ActiveRequests   int
	DisabledUntil    time.Time
	ConsecutiveFails int
	LastError        string
	CircuitBreaker   *CircuitBreaker
	Mu               sync.Mutex
}

// CircuitBreaker 断路器
type CircuitBreaker struct {
	Type      CircuitBreakerType
	StartTime time.Time
	Reason    string
}

// CircuitBreakerType 断路器类型
type CircuitBreakerType int

const (
	CircuitBreakerError = iota + 1
	CircuitBreakerMaxConn
)

// IsActive 是否激活
func (cb *CircuitBreaker) IsActive() bool {
	if cb == nil || cb.StartTime.IsZero() {
		return false
	}
	return time.Since(cb.StartTime) < 15*time.Second
}
