package storage

import (
	"sync"
	"time"

	"github.com/mohae/deepcopy"
)

// item 缓存项
type item struct {
	value      interface{}
	expireTime int64
}

// LocalCache 本地缓存
type LocalCache struct {
	data     map[string]item
	mutex    sync.RWMutex
	stopChan chan struct{}
	interval time.Duration
}

// NewLocalCache 创建一个本地缓存实例
func NewLocalCache(cleanInterval time.Duration) *LocalCache {
	c := &LocalCache{
		data:     make(map[string]item),
		stopChan: make(chan struct{}),
		interval: cleanInterval,
	}
	go c.cleanupWorker()
	return c
}

// Set 设置缓存值
func (c *LocalCache) Set(key string, value interface{}, ttl time.Duration, clone bool) {
	var expire int64
	if ttl > 0 {
		expire = time.Now().Add(ttl).UnixNano()
	}
	if clone {
		value = deepcopy.Copy(value)
	}
	c.mutex.Lock()
	c.data[key] = item{
		value:      value,
		expireTime: expire,
	}
	c.mutex.Unlock()
}

// Get 获取缓存值
func (c *LocalCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	it, found := c.data[key]
	c.mutex.RUnlock()
	if !found {
		return nil, false
	}
	if it.expireTime > 0 && time.Now().UnixNano() > it.expireTime {
		c.Delete(key)
		return nil, false
	}
	return it.value, true
}

// Exists 是否存在缓存值
func (c *LocalCache) Exists(key string) bool {
	_, found := c.Get(key)
	return found
}

// Delete 删除缓存值
func (c *LocalCache) Delete(key string) {
	c.mutex.Lock()
	delete(c.data, key)
	c.mutex.Unlock()
}

// cleanupWorker 清理过期缓存值的工作器
func (c *LocalCache) cleanupWorker() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now().UnixNano()
			c.mutex.Lock()
			for k, v := range c.data {
				if v.expireTime > 0 && now > v.expireTime {
					delete(c.data, k)
				}
			}
			c.mutex.Unlock()
		case <-c.stopChan:
			return
		}
	}
}

// Stop 停止清理过期缓存值的工作器
func (c *LocalCache) Stop() {
	close(c.stopChan)
}
