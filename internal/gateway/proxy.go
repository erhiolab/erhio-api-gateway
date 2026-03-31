package gateway

import (
	"elake-api-gateway/internal/utils"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Proxy 代理请求
func Proxy(target string, w http.ResponseWriter, r *http.Request) {
	parseURL, err := url.Parse(target)
	if err != nil {
		utils.Error(w, http.StatusBadGateway, 5020, "Bad Gateway")
		return
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(parseURL)
			// Request ID
			if rid, ok := pr.In.Context().Value(utils.RequestIDKey).(string); ok && rid != "" {
				if pr.Out.Header.Get("X-Request-ID") == "" {
					pr.Out.Header.Set("X-Request-ID", rid)
				}
			}

			// Client IP（关键）
			if ip, ok := pr.In.Context().Value(utils.ClientIPKey).(string); ok && ip != "" {
				pr.Out.Header.Set("X-Real-IP", ip)
				// 标准链路透传
				if pr.Out.Header.Get("X-Forwarded-For") == "" {
					pr.Out.Header.Set("X-Forwarded-For", ip)
				} else {
					pr.Out.Header.Set("X-Forwarded-For",
						pr.Out.Header.Get("X-Forwarded-For")+", "+ip)
				}
			}
		},
	}
	proxy.ServeHTTP(w, r)
}
