package gateway

import (
	"elake-api-gateway/internal/utils"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// Proxy 代理请求
func Proxy(target string, w http.ResponseWriter, r *http.Request) {
	parseURL, err := url.Parse(target)
	if err != nil {
		utils.BadGateway(w)
		return
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(parseURL)
			// 重写路径
			if newPath, ok := pr.In.Context().Value(utils.UpstreamPathKey).(string); ok && newPath != "" {
				pr.Out.URL.Path = singleJoiningSlash(parseURL.Path, newPath)
			}
			// 透传查询参数
			pr.Out.URL.RawQuery = pr.In.URL.RawQuery
			// Request ID
			if rid, ok := pr.In.Context().Value(utils.RequestIDKey).(string); ok && rid != "" {
				pr.Out.Header.Set("X-Request-ID", rid)
			}
			// Client IP（关键）
			if ip, ok := pr.In.Context().Value(utils.ClientIPKey).(string); ok && ip != "" {
				pr.Out.Header.Set("X-Real-IP", ip)
				if pr.Out.Header.Get("X-Forwarded-For") == "" {
					pr.Out.Header.Set("X-Forwarded-For", ip)
				} else {
					pr.Out.Header.Set("X-Forwarded-For", pr.Out.Header.Get("X-Forwarded-For")+", "+ip)
				}
			}
		},
	}
	proxy.ServeHTTP(w, r)
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
