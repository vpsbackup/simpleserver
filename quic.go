package main

import (
	"github.com/quic-go/quic-go/http3"
	"log"
	"net/http"
)

func RunHTTP3(hand http.Handler) {
	log.Println("http3 listen on:", Cfg.QuicAddr)
	err := http3.ListenAndServeQUIC(Cfg.QuicAddr, Cfg.QuicCertPath, Cfg.QuicKeyPath, hand)
	if err != nil {
		panic(err)
	}
}
