package gateway

import (
	"bytes"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/service/loadBalancer"
	"elake-api-gateway/internal/utils"
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
		utils.BadGateway(w)
		return
	}
	resetBody, err := snapshotRequestBody(r)
	if err != nil {
		logger.WithRequestLogCtx(r.Context()).Error("代理请求: 无法缓存请求体",
			zap.Int64("service_id", service.ID),
			zap.String("service_name", service.Name),
			zap.Error(err),
		)
		utils.BadGateway(w)
		return
	}
	tried := make(map[int64]struct{}, len(service.Nodes))
	current := selected.Node
	for current != nil {
		tried[current.ID] = struct{}{}
		selected.Node = current
		if err := resetBody(); err != nil {
			logger.WithRequestLogCtx(r.Context()).Error("代理请求: 无法重置请求体",
				zap.Int64("service_id", service.ID),
				zap.String("service_name", service.Name),
				zap.Error(err),
			)
			utils.BadGateway(w)
			return
		}
		proxyErr, retryable := proxyToNode(current.NodeURL, w, r)
		loadBalancer.ReleaseNodeRequest(service.ID, current.ID)
		if proxyErr == nil {
			loadBalancer.MarkNodeSuccess(service.ID, current.ID)
			return
		}
		loadBalancer.MarkNodeFailure(service.ID, current.ID, proxyErr)
		next := loadBalancer.SelectNode(service, tried)
		if !retryable || next == nil {
			logger.WithRequestLogCtx(r.Context()).Warn("代理请求: 上游节点不可用",
				zap.Int64("service_id", service.ID),
				zap.String("service_name", service.Name),
				zap.Int64("node_id", current.ID),
				zap.String("node_url", current.NodeURL),
				zap.Error(proxyErr),
			)
			if retryable {
				utils.BadGateway(w)
			}
			return
		}
		logger.WithRequestLogCtx(r.Context()).Warn("代理请求: 上游节点异常, 自动切换",
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
	utils.BadGateway(w)
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
			if rid, ok := pr.In.Context().Value(utils.RequestIDKey).(string); ok && rid != "" {
				pr.Out.Header.Set("X-Request-ID", rid)
			}
			if ip, ok := pr.In.Context().Value(utils.ClientIPKey).(string); ok && ip != "" {
				pr.Out.Header.Set("X-Real-IP", ip)
				if pr.Out.Header.Get("X-Forwarded-For") == "" {
					pr.Out.Header.Set("X-Forwarded-For", ip)
				} else {
					pr.Out.Header.Set("X-Forwarded-For", pr.Out.Header.Get("X-Forwarded-For")+", "+ip)
				}
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
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyFromEnvironment
	transport.DialContext = (&net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
	}).DialContext
	transport.TLSHandshakeTimeout = timeout
	transport.ResponseHeaderTimeout = timeout
	transport.ExpectContinueTimeout = time.Second
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
