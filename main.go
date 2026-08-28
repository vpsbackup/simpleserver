package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
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
		return
	}

	if Cfg.UseQuic {
		if Cfg.QuicOnly {
			RunHTTP3(mux)
		} else {
			go RunHTTP3(mux)
		}
	}

	addr := ":" + strconv.FormatUint(uint64(Cfg.Port), 10)
	log.Println("http1 and http2 listen on:", addr)
	err = http.ListenAndServe(addr, mux)
	if err != nil {
		log.Println("listen and serve error is:", err)
	}
}
