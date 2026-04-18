package loadBalancer

import (
	"elake-api-gateway/internal/models"
	"sync"
	"time"
)

// DBUpdater 数据库更新器接口
type DBUpdater interface {
	UpdateNodeAvailability(updates []models.NodeAvailabilityUpdate) error
}

var (
	runtimeNodes = make(map[int64][]*models.WeightedNode)
	nodesMu      sync.RWMutex
	dbUpdater    DBUpdater
)

// nodeFailureCooldown 节点故障冷却时间
const nodeFailureCooldown = 15 * time.Second

// Init 初始化负载均衡器
func Init(db DBUpdater) {
	dbUpdater = db
}
