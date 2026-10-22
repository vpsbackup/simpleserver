package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/quic-go/quic-go/http3"
)

const (
	// 只限制读请求头的耗时，不设 ReadTimeout：上传大文件和慢速客户端需要较长的 body 读取时间。
	httpReadHeaderTimeout = 10 * time.Second
	// IdleTimeout 要大于 nginx keepalive_timeout（默认 75s），否则复用的连接会被提前关掉。
	httpIdleTimeout = 90 * time.Second
	// 收到退出信号后，最多等这么久让在途请求结束。
	shutdownTimeout = 10 * time.Second
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	if len(os.Args) >= 2 && os.Args[1] == "version" {
		PrintVersionJSON()
		return
	}

	configPath := ParseArgs()
	if err := LoadConfig(configPath); err != nil {
		log.Println("load config error:", err)
		os.Exit(1)
	}

	if Cfg.UdpPort != 0 {
		go ListenUDP(Cfg.UdpPort)
	}
	if Cfg.TcpPort != 0 {
		go ListenTCP(Cfg.TcpPort)
	}

	mux, err := InitMux()
	if err != nil {
		log.Println("init mux error:", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var h3 *http3.Server
	if Cfg.UseQuic {
		h3, err = NewHTTP3Server(mux)
		if err != nil {
			log.Println("init http3 error:", err)
			os.Exit(1)
		}
	}

	srv := &http.Server{
		Addr:              ":" + strconv.FormatUint(uint64(Cfg.Port), 10),
		Handler:           mux,
		ReadHeaderTimeout: httpReadHeaderTimeout,
		IdleTimeout:       httpIdleTimeout,
	}

	var (
		wg       sync.WaitGroup
		serveErr = make(chan error, 2)
	)
	if !Cfg.QuicOnly {
		log.Println("http1 and http2 listen on:", srv.Addr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			serveErr <- srv.ListenAndServe()
		}()
	}
	if h3 != nil {
		log.Println("http3 listen on:", Cfg.QuicAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			serveErr <- h3.ListenAndServe()
		}()
	}

	// 任一 listener 异常退出，或收到 SIGINT/SIGTERM，都走同一条优雅关闭路径。
	exitCode := 0
	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println("serve error:", err)
			exitCode = 1
		}
	case <-ctx.Done():
		log.Println("shutdown signal received, draining connections")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if !Cfg.QuicOnly {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Println("http shutdown error:", err)
			srv.Close()
		}
	}
	if h3 != nil {
		if err := h3.Shutdown(shutdownCtx); err != nil {
			log.Println("http3 shutdown error:", err)
			h3.Close()
		}
	}
	wg.Wait()
	os.Exit(exitCode)
}
