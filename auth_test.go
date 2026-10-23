package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func enableTestAuth(t *testing.T) {
	t.Helper()
	old := Cfg
	oldLimiter := loginLimiter
	Cfg.AuthPassword = "site-pass-123"
	Cfg.CookiePass = "cookie-secret-456"
	Cfg.AuthExpireS = 3600
	initAuth(Cfg)
	loginLimiter = newLoginGuard()
	t.Cleanup(func() {
		Cfg = old
		initAuth(Cfg)
		loginLimiter = oldLimiter
	})
}

func TestAuthDisabledAllowsProtected(t *testing.T) {
	old := Cfg
	Cfg.AuthPassword = ""
	initAuth(Cfg)
	t.Cleanup(func() {
		Cfg = old
		initAuth(Cfg)
	})
	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	if needsAuth(req) {
		t.Fatal("auth disabled should not need auth")
	}
	if !Authorized(req) {
		t.Fatal("auth disabled should authorize")
	}
}

func TestNeedsAuthPaths(t *testing.T) {
	enableTestAuth(t)
	cases := []struct {
		method string
		path   string
		want   bool
	}{
		{"POST", "/upload", true},
		{"GET", "/upload", false},
		{"POST", "/t", true},
		{"GET", "/t", false},
		{"POST", "/api/t", true},
		{"POST", "/api/t/delete", true},
		{"POST", "/api/files/delete", true},
		{"POST", "/v1/agentproxy/chat/completions", true},
		{"GET", "/v1/agentproxy/models", true},
		{"OPTIONS", "/v1/agentproxy/chat/completions", false},
		{"GET", "/api/files/list", true},
		{"GET", "/vnstat", false},
		{"GET", "/", false},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		if got := needsAuth(req); got != tc.want {
			t.Fatalf("%s %s needsAuth=%v want %v", tc.method, tc.path, got, tc.want)
		}
	}
}

// #11: a write route that nobody registered in an allow-list must still be
// protected, while login/logout and read-only pages must stay public.
func TestNeedsAuthDefaultsToDenyWrites(t *testing.T) {
	enableTestAuth(t)
	protected := []struct{ method, path string }{
		{"POST", "/some/new/route"},
		{"PUT", "/upload"},
		{"DELETE", "/api/t/1"},
		{"PATCH", "/healthz"},
		{"POST", "/api/dns/query"},
	}
	for _, tc := range protected {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		if !needsAuth(req) {
			t.Errorf("%s %s should need auth by default", tc.method, tc.path)
		}
	}
	public := []struct{ method, path string }{
		{"POST", "/login"},
		{"POST", "/logout"},
		{"GET", "/"},
		{"GET", "/upload"},
		{"GET", "/healthz"},
		{"HEAD", "/"},
		{"OPTIONS", "/api/t"},
		{"GET", "/t"},
		{"GET", "/api/t/list"},
	}
	for _, tc := range public {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		if needsAuth(req) {
			t.Errorf("%s %s should stay public", tc.method, tc.path)
		}
	}
}

func TestLoginSetsCookieAndAuthorizes(t *testing.T) {
	enableTestAuth(t)
	form := strings.NewReader("password=site-pass-123")
	req := httptest.NewRequest(http.MethodPost, "/login", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	LoginHandler(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
	}
	cookies := rr.Result().Cookies()
	var auth *http.Cookie
	for _, c := range cookies {
		if c.Name == authCookieName {
			auth = c
		}
	}
	if auth == nil || auth.Value == "" {
		t.Fatalf("missing auth cookie: %#v", cookies)
	}
	if !auth.HttpOnly {
		t.Fatal("cookie should be HttpOnly")
	}

	req2 := httptest.NewRequest(http.MethodPost, "/upload", nil)
	req2.AddCookie(auth)
	if !Authorized(req2) {
		t.Fatal("cookie should authorize")
	}
}

func TestWrongPasswordRejected(t *testing.T) {
	enableTestAuth(t)
	form := strings.NewReader("password=wrong")
	req := httptest.NewRequest(http.MethodPost, "/login", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	LoginHandler(rr, req)
	if rr.Code != http.StatusFound || !strings.Contains(rr.Header().Get("Location"), "err=") {
		t.Fatalf("status=%d loc=%q", rr.Code, rr.Header().Get("Location"))
	}
}

func TestSiteTokenAndBearerAuth(t *testing.T) {
	enableTestAuth(t)
	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	req.Header.Set("X-Site-Token", "site-pass-123")
	if !Authorized(req) {
		t.Fatal("X-Site-Token should authorize")
	}

	req2 := httptest.NewRequest(http.MethodPost, "/upload", nil)
	req2.Header.Set("Authorization", "Bearer site-pass-123")
	if !Authorized(req2) {
		t.Fatal("Bearer site password should authorize")
	}

	req3 := httptest.NewRequest(http.MethodPost, "/v1/agentproxy/x", nil)
	req3.Header.Set("Authorization", "Bearer sk-upstream-api-key")
	if Authorized(req3) {
		t.Fatal("upstream api key must not authorize site routes without cookie")
	}
}

func TestMuxRejectsUnauthorizedWrite(t *testing.T) {
	enableTestAuth(t)
	Cfg.Msg = false
	Cfg.Tcping = false
	Cfg.EnableView = false
	Cfg.Tracer = false
	Cfg.StaticDir = "./public"
	mux, err := InitMux()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("x"))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusFound {
		bt, _ := io.ReadAll(rr.Result().Body)
		t.Fatalf("status=%d body=%q", rr.Code, bt)
	}
}

func TestConfigRejectsSameAuthSecrets(t *testing.T) {
	cfg := defaultConfig()
	cfg.AuthPassword = "same"
	cfg.CookiePass = "same"
	if err := validateConfig(cfg); err == nil {
		t.Fatal("expected validation error")
	}
}

// #21: a cookie whose embedded expiry timestamp is in the past must be
// rejected even if the signature part is correct, and fresh/tampered values
// must classify correctly.
func TestAuthCookieExpiry(t *testing.T) {
	enableTestAuth(t)

	future := time.Now().Add(3600 * time.Second).Unix()
	valid := calcAuthCookieValue(Cfg.CookiePass, authCookieKey, future)
	past := time.Now().Add(-3600 * time.Second).Unix()
	expired := calcAuthCookieValue(Cfg.CookiePass, authCookieKey, past)
	tampered := func(v string) string {
		return calcAuthCookieValue(Cfg.CookiePass, authCookieKey, future) + "x"
	}(valid)

	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{"fresh", valid, true},
		{"expired", expired, false},
		{"tampered", tampered, false},
		{"garbage", "not-a-cookie", false},
		{"missing", "", false},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if tc.value != "" {
			req.AddCookie(&http.Cookie{Name: authCookieName, Value: tc.value})
		}
		if got := cookieAuthorized(req); got != tc.want {
			t.Errorf("%s cookie: cookieAuthorized = %t, want %t", tc.name, got, tc.want)
		}
	}
}

func doLogin(t *testing.T, addr, pass string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"password": {pass}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = addr
	rr := httptest.NewRecorder()
	LoginHandler(rr, req)
	return rr
}

func hasAuthCookie(rr *httptest.ResponseRecorder) bool {
	for _, c := range rr.Result().Cookies() {
		if c.Name == authCookieName && c.Value != "" {
			return true
		}
	}
	return false
}

// #10: repeated failures from one address are throttled, a correct password
// is still refused while locked, and other clients are unaffected.
func TestLoginFailureBackoff(t *testing.T) {
	enableTestAuth(t)
	locked := "203.0.113.7:40000"

	for i := 1; i <= loginFailFree; i++ {
		rr := doLogin(t, locked, "wrong")
		if rr.Code != http.StatusFound {
			t.Fatalf("failure %d: status=%d want redirect", i, rr.Code)
		}
		if rr.Header().Get("Retry-After") != "" {
			t.Fatalf("failure %d: unexpected Retry-After before the lock", i)
		}
	}

	rr := doLogin(t, locked, "wrong")
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("after %d failures: status=%d want 429", loginFailFree, rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Fatal("429 response should carry Retry-After")
	}

	// Even the right password is rejected while the lock is active.
	rr = doLogin(t, locked, "site-pass-123")
	if rr.Code != http.StatusTooManyRequests || hasAuthCookie(rr) {
		t.Fatalf("locked client got status=%d cookie=%v", rr.Code, hasAuthCookie(rr))
	}

	// A different source address is not affected.
	rr = doLogin(t, "198.51.100.9:5555", "site-pass-123")
	if rr.Code != http.StatusFound || !hasAuthCookie(rr) {
		t.Fatalf("other client: status=%d cookie=%v", rr.Code, hasAuthCookie(rr))
	}
}

// #10: the lock expires and a successful login clears the failure history.
func TestLoginBackoffExpiresAndResets(t *testing.T) {
	enableTestAuth(t)
	ip := "203.0.113.8"
	addr := ip + ":40001"

	for i := 0; i <= loginFailFree; i++ {
		doLogin(t, addr, "wrong")
	}
	if wait := loginLimiter.retryAfter(ip); wait <= 0 {
		t.Fatalf("expected active lock, got %v", wait)
	}

	// Advance past the longest possible lock.
	loginLimiter.now = func() time.Time { return time.Now().Add(loginLockMax + time.Minute) }
	if wait := loginLimiter.retryAfter(ip); wait != 0 {
		t.Fatalf("lock should have expired, retryAfter=%v", wait)
	}
	rr := doLogin(t, addr, "site-pass-123")
	if rr.Code != http.StatusFound || !hasAuthCookie(rr) {
		t.Fatalf("after expiry: status=%d cookie=%v", rr.Code, hasAuthCookie(rr))
	}
	// Success wipes the counters.
	loginLimiter.mu.Lock()
	_, ok := loginLimiter.entries[ip]
	loginLimiter.mu.Unlock()
	if ok {
		t.Fatal("successful login should clear the failure entry")
	}

	// A fresh run of failures starts from zero again.
	for i := 1; i <= loginFailFree; i++ {
		if rr := doLogin(t, addr, "wrong"); rr.Code != http.StatusFound {
			t.Fatalf("post-reset failure %d: status=%d", i, rr.Code)
		}
	}
}

// #10: cross-site state-changing requests are rejected, same-origin and
// non-browser clients are allowed.
func TestOriginAllowed(t *testing.T) {
	enableTestAuth(t)
	Cfg.Domain = "site.example"

	cases := []struct {
		name      string
		method    string
		origin    string
		host      string
		forwarded string
		site      string
		want      bool
	}{
		{"get-foreign-origin", http.MethodGet, "https://evil.example", "site.example", "", "", true},
		{"post-no-origin", http.MethodPost, "", "site.example", "", "", true},
		{"post-same-host", http.MethodPost, "https://site.example", "site.example", "", "", true},
		{"post-same-host-port", http.MethodPost, "https://site.example:8443", "site.example", "", "", true},
		{"post-configured-domain", http.MethodPost, "https://site.example", "127.0.0.1:8080", "", "", true},
		{"post-proxy-kept-origin", http.MethodPost, "https://arm.example.com", "127.0.0.1:8080", "", "", true},
		{"post-forwarded-host", http.MethodPost, "https://arm.example.com", "10.0.0.2:8080", "arm.example.com", "", true},
		{"post-foreign-origin", http.MethodPost, "https://evil.example", "site.example", "", "", false},
		{"post-foreign-origin-public-host", http.MethodPost, "https://evil.example", "203.0.113.10:443", "", "", false},
		{"post-null-origin", http.MethodPost, "null", "site.example", "", "", false},
		{"post-cross-site-fetch", http.MethodPost, "", "site.example", "", "cross-site", false},
		{"post-same-origin-fetch", http.MethodPost, "", "site.example", "", "same-origin", true},
		{"options-foreign-origin", http.MethodOptions, "https://evil.example", "site.example", "", "", true},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, "/upload", nil)
		req.Host = tc.host
		if tc.origin != "" {
			req.Header.Set("Origin", tc.origin)
		}
		if tc.forwarded != "" {
			req.Header.Set("X-Forwarded-Host", tc.forwarded)
		}
		if tc.site != "" {
			req.Header.Set("Sec-Fetch-Site", tc.site)
		}
		if got := originAllowed(req); got != tc.want {
			t.Errorf("%s: originAllowed=%v want %v", tc.name, got, tc.want)
		}
	}
}

func TestMuxRejectsCrossOriginWrite(t *testing.T) {
	enableTestAuth(t)
	Cfg.Msg = false
	Cfg.Tcping = false
	Cfg.EnableView = false
	Cfg.Tracer = false
	Cfg.StaticDir = "./public"
	mux, err := InitMux()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("x"))
	// Send a valid site token so only the origin check can reject this.
	req.Header.Set("X-Site-Token", Cfg.AuthPassword)
	req.Header.Set("Origin", "https://evil.example")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
	}
}
