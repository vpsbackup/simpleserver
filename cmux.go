package main

import (
	"errors"
	dnet "github.com/dilfish/tools/net"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net"
	"net/http"
)

type MuxService struct {
	Mux http.ServeMux
}

func (s *MuxService) Handle(pattern string, handler http.Handler) {
	s.Mux.Handle(pattern, handler)
}

func (s *MuxService) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.Mux.HandleFunc(pattern, handler)
}

func (s *MuxService) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	isBlock := dnet.CheckBlocked(r)
	if isBlock {
		log.Println("request is blocked:", r)
		w.Write([]byte(dnet.BlockHTML))
		return
	}
	if r.RequestURI != "/metrics" {
		log.Println("request is:", r.Method, r.RequestURI)
		log.Println("headers are:", r.Header)
	}
	// for http3
	if r.TLS != nil {
		if r.TLS.ServerName != *FlagDomain {
			log.Println("bad service name:", r.TLS.ServerName)
			return
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Println("split host port error:", r.RemoteAddr, err)
	}
	if r.ProtoMajor == 3 {
		r.Header.Set("X-Real-Ip", host)
		r.Header.Set("X-HTTP3-Enable", "true")
		w.Header().Set("X-HTTP3-Enable", "true")
	} else {
		if !*FlagBehindNginx {
			r.Header.Set("X-Real-Ip", host)
		}
		r.Header.Set("X-HTTP3-Enable", "false")
		w.Header().Set("X-HTTP3-Enable", "false")
	}
	log.Println("request proto is:", r.Proto)
	s.Mux.ServeHTTP(w, r)
}

func InitMux() (*MuxService, error) {
	var mux MuxService
	mux.Mux = http.ServeMux{}

	if *FlagTcping {
		RunTcping()
		mux.Handle("/metrics", promhttp.Handler())
	}

	if *FlagEnableView {
		err := InitView(*FlagV4Fn, *FlagV6Fn)
		if err != nil {
			log.Println("bad view file")
			return nil, errors.New("bad view file")
		}
		mux.HandleFunc("/ip/", GlobalViewService.Handle)
	}

	if *FlagMsg {
		MClient = NewMongoClient("mongodb://localhost:27017", "msglist", "msg")
		if MClient == nil {
			return nil, errors.New("new mongo client error")
		}
		mux.HandleFunc("/t", GetMessagePage)
	}

	if *FlagTracer {
		mux.HandleFunc("/tracer", HandleTracer)
	}

	mux.HandleFunc("/ip", CFIPHandler)
	mux.HandleFunc("/memfile/", MemFileHandler)
	mux.HandleFunc("/vnstat", VnstatHandler)
	mux.HandleFunc("/api/", ApiHandler)
	mux.HandleFunc("/upload", Uploader)

	mux.Handle("/", http.FileServer(http.Dir("./")))

	return &mux, nil
}
