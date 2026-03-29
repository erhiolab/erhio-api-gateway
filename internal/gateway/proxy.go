package gateway

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Proxy 代理请求
func Proxy(target string, w http.ResponseWriter, r *http.Request) {
	parseURL, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(parseURL)
	proxy.ServeHTTP(w, r)
}
