package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	errUpstreamMissing  = errors.New("missing x-upstream-base header")
	errUpstreamInvalid  = errors.New("invalid x-upstream-base")
	errUpstreamNotHTTPS = errors.New("upstream must use https")
	errUpstreamNotAllow = errors.New("upstream host not allowed")
	errUpstreamPort     = errors.New("upstream port not allowed")
)

// AgentProxy 转发 OpenAI 兼容接口的请求，解决浏览器跨域问题。
// 前端通过 POST /v1/agentproxy/chat/completions 发起请求，
// 并在 header 中携带 x-upstream-base 指定真实 API 地址。
// 上游 Host 必须命中 -auh 白名单，且仅允许 https。
func AgentProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		setCORS(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	upstream := strings.TrimSpace(r.Header.Get("x-upstream-base"))
	if upstream == "" {
		http.Error(w, errUpstreamMissing.Error(), http.StatusBadRequest)
		return
	}
	u, err := url.Parse(upstream)
	if err != nil || u.Host == "" {
		http.Error(w, errUpstreamInvalid.Error(), http.StatusBadRequest)
		return
	}
	if err := allowedAgentUpstream(u); err != nil {
		status := http.StatusForbidden
		if errors.Is(err, errUpstreamNotHTTPS) || errors.Is(err, errUpstreamInvalid) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	// 移除自定义 header，避免透传到上游
	r.Header.Del("x-upstream-base")

	// 共享 Transport（review #13）：拨号/TLS 握手/响应头都带超时，上游挂死
	// 不会无限占住客户端连接；SSE 流式靠 FlushInterval 立即刷新。
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.Transport = upstreamTransport()
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = u.Scheme
		req.URL.Host = u.Host
		// 将 /v1/agentproxy/xxx 映射到上游路径（保留上游 base 自带的路径前缀）
		p := strings.TrimPrefix(req.URL.Path, "/v1/agentproxy")
		if p == "" {
			p = "/"
		}
		base := strings.TrimSuffix(u.Path, "/")
		req.URL.Path = base + p
		req.Host = u.Host
	}
	proxy.FlushInterval = 0 // 立即刷新，支持 SSE 流式输出
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("agentproxy upstream error: %v", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
	}

	setCORS(w, r)
	proxy.ServeHTTP(w, r)
}

func hostAllowed(host string) bool {
	host = strings.ToLower(host)
	if host == "" || len(AgentUpstreamHosts) == 0 {
		return false
	}
	if _, ok := AgentUpstreamHosts[host]; ok {
		return true
	}
	// 后缀匹配：白名单 heilovehei.com 可放行 cn3.heilovehei.com
	// 要求前导 '.'，避免 evilheilovehei.com 误放行
	for allowed := range AgentUpstreamHosts {
		if allowed != "" && strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}
	return false
}

func allowedAgentUpstream(u *url.URL) error {
	if u == nil || u.Host == "" {
		return errUpstreamInvalid
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return errUpstreamNotHTTPS
	}
	// 只放行 443（显式 :443 与省略端口等价），避免被当作反代跳板（review #13）；
	// 回环地址豁免，方便本机联调（白名单本来也不该写外网回环）。
	host := strings.ToLower(u.Hostname())
	if p := u.Port(); p != "" && p != "443" {
		if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
			return errUpstreamPort
		}
	}
	if host == "" {
		return errUpstreamInvalid
	}
	if !hostAllowed(host) {
		return errUpstreamNotAllow
	}
	return nil
}

// upstreamTransport returns the shared upstream Transport; tests may swap the
// package-level agentProxyTransport to inject a fake RoundTripper.
var agentProxyTransport http.RoundTripper
var agentTransportOnce sync.Once

func upstreamTransport() http.RoundTripper {
	if agentProxyTransport != nil {
		return agentProxyTransport
	}
	agentTransportOnce.Do(func() {
		agentProxyTransport = &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConnsPerHost:   16,
		}
	})
	return agentProxyTransport
}

// setCORS reflects same-origin callers instead of wildcard (review #13):
// agent.html 与本站同源，根本用不到 CORS 头；放开 * 等于让任意第三方页面
// 借道本机中继。放行本站域名及其子域、本机（localhost / 127.0.0.1）。
func setCORS(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" {
		if u, err := url.Parse(origin); err == nil {
			host := strings.ToLower(u.Hostname())
			site := strings.ToLower(Cfg.Domain)
			if (site != "" && (host == site || strings.HasSuffix(host, "."+site))) ||
				host == "localhost" || strings.HasPrefix(host, "127.0.0.1") {
				w.Header().Add("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-upstream-base, HTTP-Referer, X-Title")
}
