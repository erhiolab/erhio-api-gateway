package middleware

import (
	"context"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/utils"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

const (
	// uuidPoolSize UUID池大小
	defaultUUIDPoolSize    = 4096
	// uuidPoolLowMark UUID池低水位
	defaultUUIDPoolLowMark = defaultUUIDPoolSize / 4
)

// uuidPool UUID池
type uuidPool struct {
	mu      sync.Mutex
	pool    []string
	filling bool
	inited  bool
	size    int
	lowMark int
}

// globalUUIDPool 全局 UUID池
var globalUUIDPool = &uuidPool{}

// init 初始化 UUID池
func (p *uuidPool) init() {
	p.mu.Lock()
	if p.inited {
		p.mu.Unlock()
		return
	}
	cfg := config.Get()
	p.size = cfg.Gateway.UUIDPoolSize
	if p.size <= 0 {
		p.size = defaultUUIDPoolSize
	}
	p.lowMark = cfg.Gateway.UUIDPoolLowMark
	if p.lowMark <= 0 {
		p.lowMark = defaultUUIDPoolLowMark
	}
	p.inited = true
	p.mu.Unlock()
	p.refill()
}

// refill 填充 UUID池
func (p *uuidPool) refill() {
	p.mu.Lock()
	if p.filling {
		p.mu.Unlock()
		return
	}
	p.filling = true
	size := p.size
	p.mu.Unlock()

	buf := make([]string, size)
	for i := range buf {
		buf[i] = uuid.New().String()
	}

	p.mu.Lock()
	p.pool = append(p.pool, buf...)
	p.filling = false
	p.mu.Unlock()
}

// get 获取 UUID
func (p *uuidPool) get() string {
	if !p.inited {
		p.init()
	}
	p.mu.Lock()
	n := len(p.pool)
	if n > 0 {
		s := p.pool[n-1]
		p.pool = p.pool[:n-1]
		lowMark := p.lowMark
		low := n-1 < lowMark && !p.filling
		p.mu.Unlock()
		if low {
			go p.refill()
		}
		return s
	}
	p.mu.Unlock()
	return uuid.New().String()
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
