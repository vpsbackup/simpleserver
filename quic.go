package main

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/quic-go/quic-go/http3"
)

// NewHTTP3Server 构建 HTTP/3 server，交给 main 统一启动和优雅关闭。
// 证书与 nginx 用同一份文件，和 /healthz 的到期检查保持一致。
func NewHTTP3Server(hand http.Handler) (*http3.Server, error) {
	cert, err := tls.LoadX509KeyPair(Cfg.QuicCertPath, Cfg.QuicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load quic cert %s: %w", Cfg.QuicCertPath, err)
	}
	tlsCfg := http3.ConfigureTLSConfig(&tls.Config{Certificates: []tls.Certificate{cert}})
	return &http3.Server{
		Addr:      Cfg.QuicAddr,
		TLSConfig: tlsCfg,
		Handler:   hand,
	}, nil
}
