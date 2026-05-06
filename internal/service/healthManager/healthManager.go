package healthManager

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/service/email"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

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
	sendEmail        bool
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
func (m *Manager) Register(name string, failThreshold, successThreshold int32, sendEmail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.checkers[name]; exists {
		logger.Log.Warn("checker 已存在", zap.String("name", name))
		return
	}
	m.checkers[name] = &checker{
		name:             name,
		status:           int32(Healthy),
		failThreshold:    failThreshold,
		successThreshold: successThreshold,
		sendEmail:        sendEmail,
	}
}

// Report 报告健康检查结果
func (m *Manager) Report(name string, err error) {
	cfg := config.Get().Gateway
	m.mu.RLock()
	c, ok := m.checkers[name]
	m.mu.RUnlock()
	if !ok {
		return
	}
	if err != nil {
		atomic.AddInt32(&c.failCount, 1)
		atomic.StoreInt32(&c.successCount, 0)
		if atomic.LoadInt32(&c.failCount) >= c.failThreshold &&
			atomic.CompareAndSwapInt32(&c.status, int32(Healthy), int32(Unhealthy)) {
			failCount := atomic.LoadInt32(&c.failCount)
			atomic.StoreInt32(&c.failCount, 0)
			logger.Log.Warn("服务异常",
				zap.String("serviceName", name),
				zap.Int32("failCount", failCount),
				zap.Error(err),
			)
			if c.sendEmail {
				_ = email.SendMail(&email.MailPayload{
					To:      cfg.EmailUsername,
					Subject: fmt.Sprintf("[洱海网关 %s] 服务异常", cfg.ID),
					HTML:    true,
					Body:    fmt.Sprintf(`<div style="font-family: Arial, sans-serif; line-height:1.6;"><h2 style="color:#d93025;">🚨 服务异常告警</h2><p><strong>网关实例: </strong>%s</p><p><strong>服务名称: </strong>%s</p><p><strong>失败次数: </strong>%d</p><p><strong>时间: </strong>%s</p><hr><p style="color:#999;">请尽快排查服务状态!</p></div>`, cfg.ID, name, failCount, time.Now().Format("2006-01-02 15:04:05")),
				})
			}
		}
		return
	}
	atomic.AddInt32(&c.successCount, 1)
	atomic.StoreInt32(&c.failCount, 0)
	if atomic.LoadInt32(&c.successCount) >= c.successThreshold && atomic.CompareAndSwapInt32(&c.status, int32(Unhealthy), int32(Healthy)) {
		atomic.StoreInt32(&c.successCount, 0)
		logger.Log.Info("服务恢复", zap.String("serviceName", name))
		if c.sendEmail {
			_ = email.SendMail(&email.MailPayload{
				To:      cfg.EmailUsername,
				Subject: fmt.Sprintf("[洱海网关 %s] 服务恢复", cfg.ID),
				HTML:    true,
				Body:    fmt.Sprintf(`<div style="font-family: Arial, sans-serif; line-height:1.6;"><h2 style="color:#188038;">✅ 服务恢复通知</h2><p><strong>网关实例: </strong>%s</p><p><strong>服务名称: </strong>%s</p><p><strong>状态: </strong>已恢复正常</p><p><strong>时间: </strong>%s</p><hr><p style="color:#999;">服务已恢复, 无需进一步操作.</p></div>`, cfg.ID, name, time.Now().Format("2006-01-02 15:04:05")),
			})
		}
	}
}

// IsHealthy 检查服务是否健康
func (m *Manager) IsHealthy(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.checkers[name]
	if !ok {
		return false
	}
	return atomic.LoadInt32(&c.status) == int32(Healthy)
}
