package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHealthzHandler(t *testing.T) {
	old := Cfg
	t.Cleanup(func() { Cfg = old })
	Cfg = defaultConfig() // AuthPassword empty → auth disabled, full info
	resetHealthzCache()

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

func TestHealthzCacheAndRedaction(t *testing.T) {
	old := Cfg
	t.Cleanup(func() { Cfg = old })
	resetHealthzCache()

	// anonymous client: machine-locale fields are hidden, probe fields stay (#8)
	Cfg = defaultConfig()
	Cfg.UploadDir = t.TempDir()
	Cfg.AuthPassword = "sekrit"
	initAuth(Cfg)
	rec := httptest.NewRecorder()
	HealthzHandler(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	var info healthzInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if !info.OK || info.Vnstat == nil || info.Mem == nil {
		t.Fatalf("probe fields missing for anonymous client: ok=%t vnstat=%v mem=%v",
			info.OK, info.Vnstat, info.Mem)
	}
	if info.Host != nil || info.Disks != nil || info.Cert != nil {
		t.Fatalf("anonymous response leaks machine fields: host=%v disks=%v cert=%v",
			info.Host, info.Disks, info.Cert)
	}

	// authenticated client gets the full snapshot
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req2.Header.Set("X-Site-Token", "sekrit")
	HealthzHandler(rec2, req2)
	var full healthzInfo
	if err := json.Unmarshal(rec2.Body.Bytes(), &full); err != nil {
		t.Fatal(err)
	}
	if full.Host == nil || full.Host.Hostname == "" {
		t.Fatalf("authenticated response missing host info: %v", full.Host)
	}

	// within the TTL the snapshot is cached: changing the cert path config is
	// not visible until resetHealthzCache
	cert1 := writeTestCert(t, "one.871116.xyz", time.Now().Add(24*time.Hour))
	Cfg.QuicCertPath = cert1
	resetHealthzCache()
	req3 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req3.Header.Set("X-Site-Token", "sekrit")
	rec3 := httptest.NewRecorder()
	HealthzHandler(rec3, req3)
	var withCert healthzInfo
	if err := json.Unmarshal(rec3.Body.Bytes(), &withCert); err != nil {
		t.Fatal(err)
	}
	if withCert.Cert == nil || withCert.Cert.Subject != "one.871116.xyz" {
		t.Fatalf("expected cached cert subject one.871116.xyz, got %+v", withCert.Cert)
	}

	cert2 := writeTestCert(t, "two.871116.xyz", time.Now().Add(32*24*time.Hour))
	Cfg.QuicCertPath = cert2
	rec4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req4.Header.Set("X-Site-Token", "sekrit")
	HealthzHandler(rec4, req4)
	var stillCached healthzInfo
	if err := json.Unmarshal(rec4.Body.Bytes(), &stillCached); err != nil {
		t.Fatal(err)
	}
	if stillCached.Cert != nil && stillCached.Cert.Subject != "one.871116.xyz" {
		t.Fatalf("expected stale cached cert within TTL, got %+v", stillCached.Cert)
	}

	resetHealthzCache()
	rec5 := httptest.NewRecorder()
	req5 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req5.Header.Set("X-Site-Token", "sekrit")
	HealthzHandler(rec5, req5)
	var refreshed healthzInfo
	if err := json.Unmarshal(rec5.Body.Bytes(), &refreshed); err != nil {
		t.Fatal(err)
	}
	if refreshed.Cert == nil || refreshed.Cert.Subject != "two.871116.xyz" {
		t.Fatalf("expected refreshed cert subject two.871116.xyz, got %+v", refreshed.Cert)
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
	resetHealthzCache()
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

// writeTestCert create a self-signed cert (plus dummy key block) and return its path.
func writeTestCert(t *testing.T, cn string, notAfter time.Time) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	buf.WriteString(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})))
	buf.WriteString(string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})))
	p := filepath.Join(t.TempDir(), "cert.pem")
	if err := os.WriteFile(p, []byte(buf.String()), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCollectCert(t *testing.T) {
	old := Cfg
	t.Cleanup(func() { Cfg = old })

	// missing file
	Cfg = defaultConfig()
	Cfg.QuicCertPath = filepath.Join(t.TempDir(), "nope.pem")
	c := collectCert()
	if c == nil || c.Error == "" {
		t.Fatalf("expected error for missing cert file, got %+v", c)
	}

	// valid cert: first block is CERTIFICATE, second is the private key
	want := time.Now().Add(45 * 24 * time.Hour).Truncate(time.Second)
	Cfg.QuicCertPath = writeTestCert(t, "test.871116.xyz", want)
	c = collectCert()
	if c.Error != "" {
		t.Fatalf("unexpected error: %s", c.Error)
	}
	if c.Subject != "test.871116.xyz" {
		t.Fatalf("bad subject: %q", c.Subject)
	}
	if !c.NotAfter.Equal(want) {
		t.Fatalf("bad not_after: %v want %v", c.NotAfter, want)
	}
	if c.DaysLeft < 44 || c.DaysLeft > 45 {
		t.Fatalf("bad days_left: %d", c.DaysLeft)
	}

	// no path configured
	Cfg.QuicCertPath = ""
	c = collectCert()
	if c == nil || c.Error == "" {
		t.Fatalf("expected error for empty cert path, got %+v", c)
	}
}
