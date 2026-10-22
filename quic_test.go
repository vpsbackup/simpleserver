package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 证书缺失时要在启动阶段就报错，而不是 panic 在 goroutine 里。
func TestNewHTTP3ServerMissingCert(t *testing.T) {
	old := Cfg
	dir := t.TempDir()
	Cfg.QuicCertPath = filepath.Join(dir, "missing-cert.pem")
	Cfg.QuicKeyPath = filepath.Join(dir, "missing-key.pem")
	Cfg.QuicAddr = ":0"
	t.Cleanup(func() { Cfg = old })

	if _, err := NewHTTP3Server(http.NewServeMux()); err == nil {
		t.Fatal("expected error for missing cert files")
	}
}

func TestNewHTTP3Server(t *testing.T) {
	old := Cfg
	dir := t.TempDir()
	certPath, keyPath := writeSelfSignedCert(t, dir)
	Cfg.QuicCertPath = certPath
	Cfg.QuicKeyPath = keyPath
	Cfg.QuicAddr = "127.0.0.1:0"
	t.Cleanup(func() { Cfg = old })

	h3, err := NewHTTP3Server(http.NewServeMux())
	if err != nil {
		t.Fatalf("NewHTTP3Server: %v", err)
	}
	if h3.Addr != Cfg.QuicAddr {
		t.Fatalf("addr = %q, want %q", h3.Addr, Cfg.QuicAddr)
	}
	if h3.TLSConfig == nil || len(h3.TLSConfig.Certificates) != 1 {
		t.Fatal("tls config should carry the loaded certificate")
	}
	foundH3 := false
	for _, p := range h3.TLSConfig.NextProtos {
		if p == "h3" {
			foundH3 = true
		}
	}
	if !foundH3 {
		t.Fatalf("NextProtos = %v, want h3", h3.TLSConfig.NextProtos)
	}
}

func writeSelfSignedCert(t *testing.T, dir string) (certPath, keyPath string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	return certPath, keyPath
}
