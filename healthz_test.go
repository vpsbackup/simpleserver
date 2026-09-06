package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthzHandler(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	HealthzHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("unexpected content type: %s", ct)
	}
	var info healthzInfo
	if err := json.Unmarshal(rr.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if !info.OK {
		t.Fatal("expected ok: true")
	}
	if info.UptimeS < 0 {
		t.Fatalf("bad uptime: %d", info.UptimeS)
	}
	if info.Host == nil || info.Host.Hostname == "" {
		t.Fatal("expected host info with hostname")
	}
	if info.CPU == nil || info.CPU.Num < 1 {
		t.Fatal("expected cpu info with at least 1 core")
	}
	if info.Mem == nil || info.Mem.Total == 0 {
		t.Fatal("expected mem info with total > 0")
	}
	if info.Go == nil || info.Go.Version == "" {
		t.Fatal("expected go info with version")
	}
	// vnstat may be unavailable locally; the field itself must always be present.
	if info.Vnstat == nil {
		t.Fatal("expected vnstat field in response")
	}
}

func TestHealthzViaMux(t *testing.T) {
	old := Cfg
	Cfg = defaultConfig()
	Cfg.StaticDir = "./public"
	Cfg.Msg = false
	Cfg.Tcping = false
	Cfg.EnableView = false
	Cfg.Tracer = false
	Cfg.AuthPassword = ""
	initAuth(Cfg)
	t.Cleanup(func() {
		Cfg = old
		initAuth(Cfg)
	})
	mux, err := InitMux()
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"ok":true`) {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}
