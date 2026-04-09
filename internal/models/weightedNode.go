package models

import "sync"

// WeightedNode 平滑权重节点
type WeightedNode struct {
	Node          *ServiceNode
	CurrentWeight int
	Mu            sync.Mutex
}
