package main

import (
	"github.com/quic-go/quic-go/http3"
	"log"
	"net/http"
)

func RunHTTP3(hand http.Handler) {
	log.Println("http3 listen on:", *FlagQuicAddr)
	err := http3.ListenAndServeQUIC(*FlagQuicAddr, *FlagQuicCertPath, *FlagQuicKeyPath, hand)
	if err != nil {
		panic(err)
	}
}
