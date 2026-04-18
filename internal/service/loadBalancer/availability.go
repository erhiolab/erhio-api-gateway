package loadBalancer

import (
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"sync"
	"time"

	"go.uber.org/zap"
)

// availabilityUpdate 可用率更新请求
type availabilityUpdate struct {
	nodeID       int64
	availability float64
}

var (
	availabilityQueue = make(map[int64]*availabilityUpdate)
	queueMu           sync.RWMutex
	flushInterval     = 1 * time.Second
)

// startAvailabilityFlusher 启动后台刷新协程
func startAvailabilityFlusher() {
	go func() {
		ticker := time.NewTicker(flushInterval)
		defer ticker.Stop()
		for range ticker.C {
			flushAvailabilityQueue()
		}
	}()
}

// updateNodeAvailabilityToDB 将可用率更新加入队列
func updateNodeAvailabilityToDB(nodeID int64, availability float64) {
	if dbUpdater == nil {
		return
	}
	queueMu.Lock()
	availabilityQueue[nodeID] = &availabilityUpdate{
		nodeID:       nodeID,
		availability: availability,
	}
	queueMu.Unlock()
}

// flushAvailabilityQueue 批量刷新队列到数据库
func flushAvailabilityQueue() {
	if dbUpdater == nil {
		return
	}
	queueMu.Lock()
	if len(availabilityQueue) == 0 {
		queueMu.Unlock()
		return
	}
	updates := make([]models.NodeAvailabilityUpdate, 0, len(availabilityQueue))
	for _, update := range availabilityQueue {
		updates = append(updates, models.NodeAvailabilityUpdate{
			NodeID:       update.nodeID,
			Availability: update.availability,
		})
	}
	availabilityQueue = make(map[int64]*availabilityUpdate)
	queueMu.Unlock()
	err := dbUpdater.UpdateNodeAvailability(updates)
	if err != nil {
		logger.Log.Error("批量更新节点可用率失败", zap.Error(err))
	}
}

// init 初始化后台刷新
func init() {
	startAvailabilityFlusher()
}
