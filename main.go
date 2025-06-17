package main

import (
	"log"
	"net/http"
	"strconv"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	ParseFlag()

	if *FlagV {
		Version()
		return
	}

	if *FlagUdpPort != 0 {
		go ListenUDP(*FlagUdpPort)
	}
	if *FlagTcpPort != 0 {
		go ListenTCP(*FlagTcpPort)
	}

	mux, err := InitMux()
	if err != nil {
		log.Println("init mux error:", err)
		return
	}

	if *FlagQuicOnly {
		RunHTTP3(mux)
	} else {
		go RunHTTP3(mux)
	}

	addr := ":" + strconv.FormatUint(uint64(*FlagPort), 10)
	log.Println("http1 and http2 listen on:", addr)
	err = http.ListenAndServe(addr, mux)
	if err != nil {
		log.Println("listen and serve error is:", err)
	}
}
