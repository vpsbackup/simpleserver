package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func withAgentHosts(t *testing.T, hosts ...string) {
	t.Helper()
	old := AgentUpstreamHosts
	AgentUpstreamHosts = make(map[string]struct{}, len(hosts))
	for _, h := range hosts {
		AgentUpstreamHosts[strings.ToLower(h)] = struct{}{}
	}
	t.Cleanup(func() {
		AgentUpstreamHosts = old
	})
}

func TestParseAgentUpstreamHosts(t *testing.T) {
	got := normalizeAgentUpstreamHosts([]string{" llmapi.qiniu.com", "LLMAPI.QINIU.IO", "", " heilovehei.com "})
	want := []string{"llmapi.qiniu.com", "llmapi.qiniu.io", "heilovehei.com"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d (%v)", len(got), len(want), got)
	}
	for _, h := range want {
		if _, ok := got[h]; !ok {
			t.Fatalf("missing host %q in %v", h, got)
		}
	}
	if empty := normalizeAgentUpstreamHosts([]string{"", " "}); len(empty) != 0 {
		t.Fatalf("empty input should deny-all, got %v", empty)
	}
}

func TestHostAllowedSuffix(t *testing.T) {
	withAgentHosts(t, "heilovehei.com", "llmapi.qiniu.com")

	cases := []struct {
		host string
		want bool
	}{
		{"heilovehei.com", true},
		{"cn3.heilovehei.com", true},
		{"a.b.heilovehei.com", true},
		{"evilheilovehei.com", false},
		{"heilovehei.com.evil.com", false},
		{"llmapi.qiniu.com", true},
		{"api.llmapi.qiniu.com", true},
		{"qiniu.com", false},
		{"127.0.0.1", false},
		{"api.openai.com", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := hostAllowed(tc.host); got != tc.want {
			t.Fatalf("hostAllowed(%q)=%v want %v", tc.host, got, tc.want)
		}
	}
}

func TestAllowedAgentUpstream(t *testing.T) {
	withAgentHosts(t, "llmapi.qiniu.com", "heilovehei.com")

	cases := []struct {
		name    string
		raw     string
		wantErr error
	}{
		{name: "allow qiniu", raw: "https://llmapi.qiniu.com/v1", wantErr: nil},
		{name: "allow case insensitive host", raw: "https://LLMAPI.Qiniu.Com/v1", wantErr: nil},
		{name: "allow with port", raw: "https://llmapi.qiniu.com:443/v1", wantErr: nil},
		{name: "allow heilovehei subdomain", raw: "https://cn3.heilovehei.com", wantErr: nil},
		{name: "reject http", raw: "http://llmapi.qiniu.com/v1", wantErr: errUpstreamNotHTTPS},
		{name: "reject localhost", raw: "https://127.0.0.1/v1", wantErr: errUpstreamNotAllow},
		{name: "reject openai", raw: "https://api.openai.com/v1", wantErr: errUpstreamNotAllow},
		{name: "reject suffix spoof", raw: "https://evilheilovehei.com", wantErr: errUpstreamNotAllow},
		{name: "reject empty host", raw: "https:///v1", wantErr: errUpstreamInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := url.Parse(tc.raw)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got := allowedAgentUpstream(u)
			if tc.wantErr == nil {
				if got != nil {
					t.Fatalf("got %v, want nil", got)
				}
				return
			}
			if got == nil || got.Error() != tc.wantErr.Error() {
				t.Fatalf("got %v, want %v", got, tc.wantErr)
			}
		})
	}

	t.Run("empty whitelist denies all", func(t *testing.T) {
		withAgentHosts(t)
		u, _ := url.Parse("https://llmapi.qiniu.com/v1")
		if err := allowedAgentUpstream(u); err != errUpstreamNotAllow {
			t.Fatalf("got %v, want %v", err, errUpstreamNotAllow)
		}
	})
}

func TestAgentProxyRejects(t *testing.T) {
	withAgentHosts(t, "llmapi.qiniu.com", "heilovehei.com")

	cases := []struct {
		name       string
		upstream   string
		wantStatus int
		wantBody   string
	}{
		{name: "missing", upstream: "", wantStatus: http.StatusBadRequest, wantBody: errUpstreamMissing.Error()},
		{name: "invalid", upstream: "://bad", wantStatus: http.StatusBadRequest, wantBody: errUpstreamInvalid.Error()},
		{name: "http", upstream: "http://llmapi.qiniu.com/v1", wantStatus: http.StatusBadRequest, wantBody: errUpstreamNotHTTPS.Error()},
		{name: "not allowed", upstream: "https://api.openai.com/v1", wantStatus: http.StatusForbidden, wantBody: errUpstreamNotAllow.Error()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/agentproxy/chat/completions", strings.NewReader(`{}`))
			if tc.upstream != "" {
				req.Header.Set("x-upstream-base", tc.upstream)
			}
			rr := httptest.NewRecorder()
			AgentProxy(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status=%d want %d body=%q", rr.Code, tc.wantStatus, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), tc.wantBody) {
				t.Fatalf("body=%q want contains %q", rr.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestAgentProxyPathRewriteAndHeaderStrip(t *testing.T) {
	withAgentHosts(t, "example.test")

	var sawPath, sawAuth, sawUpstreamHeader, sawHost string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawPath = r.URL.Path
		sawAuth = r.Header.Get("Authorization")
		sawUpstreamHeader = r.Header.Get("x-upstream-base")
		sawHost = r.Host
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":true}`)
	}))
	defer upstream.Close()

	u, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	// httptest is http://; temporarily allow http for this integration-style rewrite check
	// by calling Director logic after an allowlisted https host check separately.
	AgentUpstreamHosts[strings.ToLower(u.Hostname())] = struct{}{}

	req := httptest.NewRequest(http.MethodGet, "http://proxy.local/v1/agentproxy/models", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("x-upstream-base", "https://"+u.Host+"/v1")

	// Bypass https-only for local httptest by validating host first, then rewriting scheme to http for dial.
	checkURL, _ := url.Parse("https://" + u.Host + "/v1")
	if err := allowedAgentUpstream(checkURL); err != nil {
		t.Fatalf("allow check: %v", err)
	}

	// Exercise the real handler path with a transport that rewrites https->http to httptest.
	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme = "http"
		r.URL.Host = u.Host
		return oldTransport.RoundTrip(r)
	})
	t.Cleanup(func() { http.DefaultTransport = oldTransport })

	rr := httptest.NewRecorder()
	AgentProxy(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
	}
	if sawPath != "/v1/models" {
		t.Fatalf("path=%q want /v1/models", sawPath)
	}
	if sawAuth != "Bearer test-token" {
		t.Fatalf("auth=%q", sawAuth)
	}
	if sawUpstreamHeader != "" {
		t.Fatalf("x-upstream-base leaked: %q", sawUpstreamHeader)
	}
	if sawHost != u.Host {
		t.Fatalf("host=%q want %q", sawHost, u.Host)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAgentProxyOptionsCORS(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/v1/agentproxy/chat/completions", nil)
	rr := httptest.NewRecorder()
	AgentProxy(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing CORS origin")
	}
}

func TestAgentHTMLDefaultUpstreamAllowed(t *testing.T) {
	// 与 config 默认值 / agent.html 默认 apiBase 对齐
	AgentUpstreamHosts = normalizeAgentUpstreamHosts([]string{"llmapi.qiniu.com", "llmapi.qiniu.io", "heilovehei.com"})
	u, err := url.Parse("https://cn3.heilovehei.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := allowedAgentUpstream(u); err != nil {
		t.Fatalf("agent.html default upstream should be allowed: %v", err)
	}
}
