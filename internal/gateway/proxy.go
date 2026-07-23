package gateway

import (
	"bytes"
	"context"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/service/loadBalancer"
	"elake-api-gateway/internal/utils"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	upstreamTransport     *http.Transport
	upstreamTransportOnce sync.Once
)

// Proxy 代理请求, 节点失败时自动切换到其他可用节点
func Proxy(service *models.Service, selected *models.SelectedNode, w http.ResponseWriter, r *http.Request) {
	if service == nil || selected == nil || selected.Node == nil {
		utils.BadGateway(w, "未选择服务节点")
		return
	}
	resetBody, err := snapshotRequestBody(r)
	if err != nil {
		logger.WithRequestLogCtx(r.Context(), r).Error("代理请求: 无法缓存请求体",
			zap.Int64("service_id", service.ID),
			zap.String("service_name", service.Name),
			zap.Error(err),
		)
		utils.BadGateway(w, "无法缓存请求体")
		return
	}
	cfg := config.Get()
	// 节点超时时间
	nodeTimeout := 3 * time.Second
	if cfg.DatabaseConfig.NodeTimeout > 0 {
		nodeTimeout = time.Duration(cfg.DatabaseConfig.NodeTimeout) * time.Second
	}
	// 总超时时间
	maxTotalTimeout := 10 * time.Second
	if cfg.DatabaseConfig.TotalTimeout > 0 {
		maxTotalTimeout = time.Duration(cfg.DatabaseConfig.TotalTimeout) * time.Second
	}
	startTime := time.Now()
	tried := make(map[int64]struct{}, len(service.Nodes))
	current := selected.Node
	for current != nil {
		if time.Since(startTime) > maxTotalTimeout {
			logger.WithRequestLogCtx(r.Context(), r).Warn("代理请求: 总超时限制, 停止尝试更多节点",
				zap.Int64("service_id", service.ID),
				zap.String("service_name", service.Name),
				zap.Duration("elapsed", time.Since(startTime)),
				zap.Duration("max_timeout", maxTotalTimeout),
				zap.Int("tried_nodes", len(tried)),
			)
			utils.BadGateway(w, "已超时, 停止尝试更多节点")
			return
		}
		tried[current.ID] = struct{}{}
		selected.Node = current
		if err := resetBody(); err != nil {
			logger.WithRequestLogCtx(r.Context(), r).Error("代理请求: 无法重置请求体",
				zap.Int64("service_id", service.ID),
				zap.String("service_name", service.Name),
				zap.Error(err),
			)
			utils.BadGateway(w, "无法重置请求体")
			return
		}
		nodeCtx, nodeCancel := context.WithTimeout(r.Context(), nodeTimeout)
		proxyErr, retryable := proxyToNode(current.NodeURL, w, r.WithContext(nodeCtx))
		nodeCancel()
		loadBalancer.ReleaseNodeRequest(service.ID, current.ID)
		if proxyErr == nil {
			loadBalancer.MarkNodeSuccess(service.ID, current.ID)
			return
		}
		if isContextTimeout(proxyErr) {
			loadBalancer.MarkNodeMaxConn(service.ID, current.ID)
		} else {
			loadBalancer.MarkNodeFailure(service.ID, current.ID, proxyErr)
		}
		next := loadBalancer.SelectNode(service, tried)
		if !retryable || next == nil {
			logger.WithRequestLogCtx(r.Context(), r).Warn("代理请求: 上游节点不可用",
				zap.Int64("service_id", service.ID),
				zap.String("service_name", service.Name),
				zap.Int64("node_id", current.ID),
				zap.String("node_url", current.NodeURL),
				zap.Error(proxyErr),
			)
			if retryable {
				utils.BadGateway(w, "上游节点不可用")
			}
			return
		}
		logger.WithRequestLogCtx(r.Context(), r).Warn("代理请求: 上游节点异常, 自动切换",
			zap.Int64("service_id", service.ID),
			zap.String("service_name", service.Name),
			zap.Int64("from_node_id", current.ID),
			zap.String("from_node_url", current.NodeURL),
			zap.Int64("to_node_id", next.ID),
			zap.String("to_node_url", next.NodeURL),
			zap.Error(proxyErr),
		)
		current = next
	}
	utils.BadGateway(w, "所有节点都不可用")
}

// proxyToNode 代理请求到目标节点
func proxyToNode(target string, w http.ResponseWriter, r *http.Request) (error, bool) {
	parseURL, err := url.Parse(target)
	if err != nil {
		return err, true
	}
	retryWriter := &retryAwareResponseWriter{ResponseWriter: w}
	var proxyErr error
	proxy := &httputil.ReverseProxy{
		Transport: getUpstreamTransport(),
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(parseURL)
			if newPath, ok := pr.In.Context().Value(utils.UpstreamPathKey).(string); ok && newPath != "" {
				pr.Out.URL.Path = singleJoiningSlash(parseURL.Path, newPath)
			}
			pr.Out.URL.RawQuery = pr.In.URL.RawQuery
			ctx := pr.In.Context()
			// 传递 request_id
			if rid, ok := ctx.Value(utils.RequestIDKey).(string); ok && rid != "" {
				pr.Out.Header.Set("X-Request-ID", rid)
			}
			// 传递客户端 IP 和 IP 位置信息
			if loc, ok := ctx.Value(utils.ClientIPKey).(*models.IPLocation); ok && loc != nil {
				pr.Out.Header.Set("X-Real-IP", loc.IP)
				if pr.Out.Header.Get("X-Forwarded-For") == "" {
					pr.Out.Header.Set("X-Forwarded-For", loc.IP)
				} else {
					pr.Out.Header.Set("X-Forwarded-For", pr.Out.Header.Get("X-Forwarded-For")+", "+loc.IP)
				}
				pr.Out.Header.Set("G-Country-Short", loc.CountryShort)
				pr.Out.Header.Set("G-Country-Long", loc.CountryLong)
				pr.Out.Header.Set("G-Region", loc.Region)
				pr.Out.Header.Set("G-City", loc.City)
				pr.Out.Header.Set("G-Latitude", fmt.Sprintf("%f", loc.Latitude))
				pr.Out.Header.Set("G-Longitude", fmt.Sprintf("%f", loc.Longitude))
				pr.Out.Header.Set("G-Zipcode", loc.Zipcode)
				pr.Out.Header.Set("G-Timezone", loc.Timezone)
			}
			// 传递 UserAgent 信息
			if ua := utils.GetUserAgentInfo(pr.In); ua != nil {
				pr.Out.Header.Set("User-Agent", ua.UserAgent)
				pr.Out.Header.Set("G-Device", ua.Device)
			}
			// 传递 Origin
			if origin := pr.In.Header.Get("Origin"); origin != "" {
				pr.Out.Header.Set("Origin", origin)
			}
		},
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
			proxyErr = err
		},
	}
	proxy.ServeHTTP(retryWriter, r)
	if proxyErr != nil {
		return proxyErr, !retryWriter.wrote
	}
	return nil, false
}

// isContextTimeout 是否上下文超时
func isContextTimeout(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "context deadline exceeded") ||
		strings.Contains(err.Error(), "context canceled")
}

// singleJoiningSlash 合并路径, 确保只有一个斜杠
func singleJoiningSlash(a, b string) string {
	aSlash := strings.HasSuffix(a, "/")
	bSlash := strings.HasPrefix(b, "/")
	switch {
	case aSlash && bSlash:
		return a + b[1:]
	case !aSlash && !bSlash:
		return a + "/" + b
	}
	return a + b
}

// snapshotRequestBody 缓存请求体
func snapshotRequestBody(r *http.Request) (func() error, error) {
	if r.Body == nil || r.Body == http.NoBody {
		return func() error { return nil }, nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if err := r.Body.Close(); err != nil {
		return nil, err
	}
	r.GetBody = func() (io.ReadCloser, error) {
		if len(body) == 0 {
			return http.NoBody, nil
		}
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	return func() error {
		if len(body) == 0 {
			r.Body = http.NoBody
			r.ContentLength = 0
			return nil
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
		return nil
	}, nil
}

// newUpstreamTransport 创建上游传输
func newUpstreamTransport() *http.Transport {
	cfg := config.Get()
	timeout := 2 * time.Second
	if cfg.Redis.DialTimeout > 0 {
		timeout = time.Duration(cfg.Redis.DialTimeout) * time.Second
	}
	maxIdleConns := cfg.Gateway.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 500
	}
	maxIdleConnsPerHost := cfg.Gateway.MaxIdleConnsPerHost
	if maxIdleConnsPerHost <= 0 {
		maxIdleConnsPerHost = 100
	}
	idleConnTimeout := cfg.Gateway.IdleConnTimeout
	if idleConnTimeout <= 0 {
		idleConnTimeout = 90
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyFromEnvironment
	transport.DialContext = (&net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
	}).DialContext
	transport.MaxIdleConns = maxIdleConns
	transport.MaxIdleConnsPerHost = maxIdleConnsPerHost
	transport.IdleConnTimeout = time.Duration(idleConnTimeout) * time.Second
	transport.TLSHandshakeTimeout = timeout
	transport.ResponseHeaderTimeout = timeout
	transport.ExpectContinueTimeout = time.Second
	transport.DisableKeepAlives = false
	return transport
}

// getUpstreamTransport 获取上游传输
func getUpstreamTransport() *http.Transport {
	upstreamTransportOnce.Do(func() {
		upstreamTransport = newUpstreamTransport()
	})
	return upstreamTransport
}

// retryAwareResponseWriter 重试感知响应写入器
type retryAwareResponseWriter struct {
	http.ResponseWriter
	wrote bool
}

// WriteHeader 写入响应头
func (w *retryAwareResponseWriter) WriteHeader(statusCode int) {
	w.wrote = true
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write 写入响应体
func (w *retryAwareResponseWriter) Write(b []byte) (int, error) {
	w.wrote = true
	return w.ResponseWriter.Write(b)
}
