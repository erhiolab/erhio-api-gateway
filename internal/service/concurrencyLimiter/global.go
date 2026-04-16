package concurrencyLimiter

var globalLimiter *ConcurrencyLimiter

// Init 初始化
func Init() {
	globalLimiter = NewConcurrencyLimiter()
}

// Get 获取全局 limiter
func Get() *ConcurrencyLimiter {
	return globalLimiter
}
