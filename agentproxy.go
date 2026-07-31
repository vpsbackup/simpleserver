package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// AgentProxy 转发 OpenAI 兼容接口的请求，解决浏览器跨域问题。
// 前端通过 POST /v1/agentproxy/chat/completions 发起请求，
// 并在 header 中携带 x-upstream-base 指定真实 API 地址。
func AgentProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		setCORS(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	upstream := strings.TrimSpace(r.Header.Get("x-upstream-base"))
	if upstream == "" {
		http.Error(w, "missing x-upstream-base header", http.StatusBadRequest)
		return
	}
	u, err := url.Parse(upstream)
	if err != nil || u.Host == "" {
		http.Error(w, "invalid x-upstream-base", http.StatusBadRequest)
		return
	}

	// 移除自定义 header，避免透传到上游
	r.Header.Del("x-upstream-base")

	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = u.Scheme
		req.URL.Host = u.Host
		// 将 /v1/agentproxy/xxx 映射到上游 /xxx
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/v1/agentproxy")
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
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

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-upstream-base")
}
