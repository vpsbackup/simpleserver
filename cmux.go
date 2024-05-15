package main

import (
	dnet "github.com/dilfish/tools/net"
	"log"
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
		w.Write([]byte(dnet.BlockHTML))
		return
	}
	if r.RequestURI != "/metrics" {
		log.Println("request is:", r.Method, r.RequestURI)
		log.Println("headers are:", r.Header)
	}
	s.Mux.ServeHTTP(w, r)
}
