package main

import (
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

var (
	errUpstreamMissing   = errors.New("missing x-upstream-base header")
	errUpstreamInvalid   = errors.New("invalid x-upstream-base")
	errUpstreamNotHTTPS  = errors.New("upstream must use https")
	errUpstreamNotAllow  = errors.New("upstream host not allowed")
)

// AgentProxy 转发 OpenAI 兼容接口的请求，解决浏览器跨域问题。
// 前端通过 POST /v1/agentproxy/chat/completions 发起请求，
// 并在 header 中携带 x-upstream-base 指定真实 API 地址。
// 上游 Host 必须命中 -auh 白名单，且仅允许 https。
func AgentProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		setCORS(w)
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

	proxy := httputil.NewSingleHostReverseProxy(u)
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
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
	}

	setCORS(w)
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
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return errUpstreamInvalid
	}
	if !hostAllowed(host) {
		return errUpstreamNotAllow
	}
	return nil
}

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-upstream-base")
}
