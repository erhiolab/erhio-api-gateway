package healthManager

import (
	"elake-api-gateway/internal/logger"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

// status 健康状态
type status int32

const (
	Healthy status = iota
	Unhealthy
)

// checker 健康检查器
type checker struct {
	name             string
	status           int32
	failCount        int32
	successCount     int32
	failThreshold    int32
	successThreshold int32
}

// Manager 健康管理器
type Manager struct {
	checkers map[string]*checker
	mu       sync.RWMutex
}

// global 全局健康管理器
var global = newManager()

// Global 获取全局健康管理器
func Global() *Manager {
	return global
}

// newManager 创建新的健康管理器
func newManager() *Manager {
	return &Manager{
		checkers: make(map[string]*checker),
	}
}

// Register 注册健康检查器
func (m *Manager) Register(name string, failThreshold, successThreshold int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkers[name] = &checker{
		name:             name,
		status:           int32(Healthy),
		failThreshold:    failThreshold,
		successThreshold: successThreshold,
	}
}

// Report 报告健康检查结果
func (m *Manager) Report(name string, err error) {
	m.mu.RLock()
	c, ok := m.checkers[name]
	m.mu.RUnlock()
	if !ok {
		return
	}
	if err != nil {
		atomic.AddInt32(&c.failCount, 1)
		atomic.StoreInt32(&c.successCount, 0)
		if c.failCount >= c.failThreshold &&
			atomic.LoadInt32(&c.status) == int32(Healthy) {
			logger.Log.Warn(name+" 服务异常", zap.Int32("failCount", c.failCount), zap.Error(err))
			atomic.StoreInt32(&c.status, int32(Unhealthy))
		}
		return
	}
	atomic.AddInt32(&c.successCount, 1)
	atomic.StoreInt32(&c.failCount, 0)
	if c.successCount >= c.successThreshold &&
		atomic.LoadInt32(&c.status) == int32(Unhealthy) {
		logger.Log.Info(name + " 服务恢复")
		atomic.StoreInt32(&c.status, int32(Healthy))
	}
}

// IsHealthy 检查服务是否健康
func (m *Manager) IsHealthy(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.checkers[name]
	if !ok {
		return true
	}
	return atomic.LoadInt32(&c.status) == int32(Healthy)
}
