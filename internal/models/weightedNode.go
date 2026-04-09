package models

import (
	"sync"
	"time"
)

// WeightedNode 平滑权重节点
type WeightedNode struct {
	Node             *ServiceNode
	CurrentWeight    int
	DisabledUntil    time.Time
	ConsecutiveFails int
	LastError        string
	Mu               sync.Mutex
}
