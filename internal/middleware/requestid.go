package middleware

import (
	"context"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/utils"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

// uuidPoolSize UUID池大小
const defaultUUIDPoolSize = 4096

// uuidPool UUID池
type uuidPool struct {
	ch   chan string
	once sync.Once
	size int
}

// globalUUIDPool 全局 UUID池
var globalUUIDPool = &uuidPool{}

// initPool 初始化池子并启动后台生产者
func (p *uuidPool) initPool() {
	p.once.Do(func() {
		cfg := config.Get()
		p.size = cfg.Gateway.UUIDPoolSize
		if p.size <= 0 {
			p.size = defaultUUIDPoolSize
		}
		p.ch = make(chan string, p.size)
		go func() {
			for {
				p.ch <- uuid.New().String()
			}
		}()
	})
}

// get 获取 UUID
func (p *uuidPool) get() string {
	p.initPool()
	select {
	case id := <-p.ch:
		return id
	default:
		// 降级为实时生成
		return uuid.New().String()
	}
}

// TraceID 追溯ID插件
func TraceID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := globalUUIDPool.get()
			ctx := context.WithValue(r.Context(), utils.RequestIDKey, requestID)
			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
