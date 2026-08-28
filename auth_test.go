package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func enableTestAuth(t *testing.T) {
	t.Helper()
	old := Cfg
	Cfg.AuthPassword = "site-pass-123"
	Cfg.CookiePass = "cookie-secret-456"
	Cfg.AuthExpireS = 3600
	initAuth(Cfg)
	t.Cleanup(func() {
		Cfg = old
		initAuth(Cfg)
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
		{"POST", "/v1/agentproxy/chat/completions", true},
		{"GET", "/v1/agentproxy/models", true},
		{"OPTIONS", "/v1/agentproxy/chat/completions", false},
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
