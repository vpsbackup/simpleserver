package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"os"

	dnet "github.com/dilfish/tools/net"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	if r.RequestURI != "/metrics" && r.RequestURI != "/healthz" {
		ip := r.Header["X-Real-Ip"]
		view := "未知"
		if len(ip) != 0 && ip[0] != "" && GlobalViewService != nil {
			view, _ = GlobalViewService.Find(ip[0])
		}
		ipShow := ""
		if len(ip) != 0 {
			ipShow = ip[0]
		}
		log.Println("request is:", r.Method, r.RequestURI, ipShow, view)
		log.Println("headers are:", redactHeaders(r.Header))
	}
	// for http3
	if r.TLS != nil {
		if r.TLS.ServerName != Cfg.Domain {
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
		if !Cfg.BehindNginx {
			r.Header.Set("X-Real-Ip", host)
		}
		r.Header.Set("X-HTTP3-Enable", "false")
		w.Header().Set("X-HTTP3-Enable", "false")
	}
	log.Println("request proto is:", r.Proto)
	if needsAuth(r) && !Authorized(r) {
		rejectUnauthorized(w, r)
		return
	}
	s.Mux.ServeHTTP(w, r)
}

func InitMux() (*MuxService, error) {
	var mux MuxService
	mux.Mux = http.ServeMux{}

	if Cfg.Tcping {
		RunTcping()
		mux.Handle("/metrics", promhttp.Handler())
	}

	if Cfg.EnableView {
		err := InitView(Cfg.V4Fn, Cfg.V6Fn)
		if err != nil {
			log.Println("bad view file")
			return nil, errors.New("bad view file")
		}
		mux.HandleFunc("/ip/", GlobalViewService.Handle)
	}

	if Cfg.Msg {
		MClient = NewMongoClient(Cfg.MongoURI, Cfg.MongoDB, Cfg.MongoColl)
		if MClient == nil {
			return nil, errors.New("new mongo client error")
		}
		mux.HandleFunc("/t", CreateMsg)
		mux.HandleFunc("/t/list", MsgListRedirect)
		mux.HandleFunc("/t/list/", MsgShow)
	}

	if Cfg.Tracer {
		mux.HandleFunc("/tracer", HandleTracer)
	}

	mux.HandleFunc("/ip", CFIPHandler)
	mux.HandleFunc("/memfile/", MemFileHandler)
	mux.HandleFunc("/vnstat", VnstatHandler)
	mux.HandleFunc("/api/", ApiHandler)
	mux.HandleFunc("/upload", Uploader)
	mux.HandleFunc("/v1/agentproxy/", AgentProxy)
	mux.HandleFunc("/login", LoginHandler)
	mux.HandleFunc("/logout", LogoutHandler)
	mux.HandleFunc("/dilfish.html", DilfishHandler)
	mux.HandleFunc("/healthz", HealthzHandler)

	staticDir := Cfg.StaticDir
	if staticDir == "" {
		staticDir = "./public"
	}
	st, err := os.Stat(staticDir)
	if err != nil {
		log.Println("static_dir error:", staticDir, err)
		return nil, errors.New("bad static_dir")
	}
	if !st.IsDir() {
		log.Println("static_dir is not a directory:", staticDir)
		return nil, errors.New("bad static_dir")
	}
	mux.Handle("/", http.FileServer(http.Dir(staticDir)))

	return &mux, nil
}
