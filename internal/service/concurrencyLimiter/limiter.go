package concurrencyLimiter

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConcurrencyLimiter 并发限制器
type ConcurrencyLimiter struct {
	currentConcurrency int32
	maxConcurrency     int32
	mu                 sync.Mutex
	ticker             *time.Ticker
	stopCh             chan struct{}
}

// NewConcurrencyLimiter 创建一个新的并发限制器
func NewConcurrencyLimiter() *ConcurrencyLimiter {
	cfg := config.Get()
	limiter := &ConcurrencyLimiter{
		currentConcurrency: 0,
		maxConcurrency:     int32(runtime.NumCPU() * cfg.Gateway.MaxConcurrencyPerCPU),
		ticker:             time.NewTicker(time.Duration(cfg.Gateway.RecalculateInterval) * time.Second),
		stopCh:             make(chan struct{}),
	}
	go limiter.autoAdjust()
	return limiter
}

// 自动调节协程
func (c *ConcurrencyLimiter) autoAdjust() {
	// 初始化时计算一次
	c.recalculateMaxConcurrency()
	// 开始定时计算
	for {
		select {
		case <-c.ticker.C:
			c.recalculateMaxConcurrency()
		case <-c.stopCh:
			c.ticker.Stop()
			return
		}
	}
}

// Stop 停止后台协程
func (c *ConcurrencyLimiter) Stop() {
	close(c.stopCh)
}

// recalculateMaxConcurrency 重新计算并发上限
func (c *ConcurrencyLimiter) recalculateMaxConcurrency() {
	c.mu.Lock()
	defer c.mu.Unlock()
	cpuCount := runtime.NumCPU()
	// 基础并发
	cfg := config.Get()
	base := int32(cpuCount * cfg.Gateway.MaxConcurrencyPerCPU)
	// 当前使用率
	var usageRatio float64
	if c.maxConcurrency > 0 {
		usageRatio = float64(c.currentConcurrency) / float64(c.maxConcurrency)
	}
	newMax := base
	// 动态调节
	switch {
	case usageRatio > 0.8:
		newMax = int32(float64(base) * 0.8)
	case usageRatio < 0.3:
		newMax = int32(float64(base) * 1.2)
	}
	// 下限保护
	minConcurrency := int32(cpuCount * 50)
	if newMax < minConcurrency {
		newMax = minConcurrency
	}
	// 没变化直接返回
	if newMax == c.maxConcurrency {
		return
	}
	oldMax := c.maxConcurrency
	c.maxConcurrency = newMax
	logger.Log.Info("更新并发上限",
		zap.Int("cpu_count", cpuCount),
		zap.Int32("old_max", oldMax),
		zap.Int32("new_max", newMax),
		zap.Float64("usage_ratio", usageRatio),
	)
}

// Acquire 获取许可
func (c *ConcurrencyLimiter) Acquire() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.currentConcurrency >= c.maxConcurrency {
		return false
	}
	c.currentConcurrency++
	return true
}

// Release 释放许可
func (c *ConcurrencyLimiter) Release() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.currentConcurrency > 0 {
		c.currentConcurrency--
	}
}

// GetCurrentConcurrency 当前并发
func (c *ConcurrencyLimiter) GetCurrentConcurrency() int32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentConcurrency
}

// GetMaxConcurrency 最大并发
func (c *ConcurrencyLimiter) GetMaxConcurrency() int32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.maxConcurrency
}
