package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigValues(t *testing.T) {
	cfg := defaultConfig()
	if !cfg.BehindNginx || !cfg.UseQuic {
		t.Fatalf("expected behind_nginx/use_quic true by default")
	}
	if cfg.Port != 10080 || cfg.Domain == "" {
		t.Fatalf("unexpected defaults: port=%d domain=%q", cfg.Port, cfg.Domain)
	}
	if len(cfg.AgentUpstreamHosts) != 3 {
		t.Fatalf("default upstream hosts=%v", cfg.AgentUpstreamHosts)
	}
}

func TestLoadConfigOmitKeysKeepDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "partial.json")
	content := []byte(`{"port": 20080, "domain": "partial.example.com"}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadConfig(path); err != nil {
		t.Fatal(err)
	}
	if Cfg.Port != 20080 {
		t.Fatalf("port=%d", Cfg.Port)
	}
	if Cfg.Domain != "partial.example.com" {
		t.Fatalf("domain=%q", Cfg.Domain)
	}
	if !Cfg.BehindNginx || !Cfg.UseQuic {
		t.Fatalf("omitted bool defaults lost: behind=%v quic=%v", Cfg.BehindNginx, Cfg.UseQuic)
	}
	if MaxHTTPPayload != 100*1024*1024 {
		t.Fatalf("MaxHTTPPayload=%d", MaxHTTPPayload)
	}
	if !hostAllowed("cn3.heilovehei.com") {
		t.Fatalf("default upstream suffix should allow cn3.heilovehei.com")
	}
}

func TestLoadConfigEmptyUpstreamDeniesAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deny.json")
	content := []byte(`{"domain":"x.example.com","agent_upstream_hosts":[]}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadConfig(path); err != nil {
		t.Fatal(err)
	}
	if len(AgentUpstreamHosts) != 0 {
		t.Fatalf("want empty allowlist, got %v", AgentUpstreamHosts)
	}
	if hostAllowed("llmapi.qiniu.com") {
		t.Fatalf("empty allowlist should deny")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(`{`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadConfig(path); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	if err := LoadConfig(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestLoadConfigValidatePort(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "badport.json")
	if err := os.WriteFile(path, []byte(`{"port": 70000, "domain":"x.example.com"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadConfig(path); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoadSampleConfig(t *testing.T) {
	if err := LoadConfig("sample.config.json"); err != nil {
		t.Fatal(err)
	}
	if Cfg.Domain != "demo.example.com" {
		t.Fatalf("domain=%q", Cfg.Domain)
	}
	if Cfg.Port != 18080 || Cfg.UseQuic {
		t.Fatalf("sample values not applied: port=%d use_quic=%v", Cfg.Port, Cfg.UseQuic)
	}
	if MaxHTTPPayload != 50*1024*1024 {
		t.Fatalf("MaxHTTPPayload=%d", MaxHTTPPayload)
	}
	if Cfg.UploadDir == "" || Cfg.MongoURI == "" {
		t.Fatalf("upload/mongo missing")
	}
	if Cfg.StaticDir != "./public" {
		t.Fatalf("static_dir=%q", Cfg.StaticDir)
	}
}

func TestStaticDirDoesNotServeRepoRoot(t *testing.T) {
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
	// agent.html is in public/
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/agent.html", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("agent.html status=%d", rr.Code)
	}
	// go.mod must not be served from repo root anymore
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, httptest.NewRequest(http.MethodGet, "/go.mod", nil))
	if rr2.Code == http.StatusOK {
		t.Fatalf("go.mod should not be publicly served, status=%d", rr2.Code)
	}
}

func TestNormalizeAgentUpstreamHosts(t *testing.T) {
	got := normalizeAgentUpstreamHosts([]string{" llmapi.qiniu.com", "LLMAPI.QINIU.IO", "", " heilovehei.com "})
	for _, h := range []string{"llmapi.qiniu.com", "llmapi.qiniu.io", "heilovehei.com"} {
		if _, ok := got[h]; !ok {
			t.Fatalf("missing %q in %v", h, got)
		}
	}
	if len(normalizeAgentUpstreamHosts(nil)) != 0 {
		t.Fatal("nil should become empty map")
	}
}
