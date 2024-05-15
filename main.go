package main

import (
	"flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strconv"
)

var FlagBehindNginx = flag.Bool("bn", true, "behind nginx")
var FlagPort = flag.Int("p", 10080, "port number")
var FlagDomain = flag.String("d", "lig.dev.ug", "domain name in upload")
var FlagTcping = flag.Bool("t", false, "tcping")
var FlagMsg = flag.Bool("m", false, "message service")
var FlagTcping6 = flag.Bool("t6", true, "tcping IPv6")
var FlagV = flag.Bool("v", false, "version")
var FlagMock = flag.Bool("mock", false, "is mock")
var FlagTracer = flag.Bool("tr", false, "enable tracer")

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.Println("BehindNginx:", *FlagBehindNginx)
	log.Println("Port:", *FlagPort)
	log.Println("Domain:", *FlagDomain)
	log.Println("Tcping:", *FlagTcping)
	log.Println("Msg:", *FlagMsg)
	log.Println("Tcping6:", *FlagTcping6)
	log.Println("V:", *FlagV)
	log.Println("IsMock:", *FlagMock)

	if *FlagV {
		Version()
		return
	}

	var mux MuxService
	mux.Mux = http.ServeMux{}

	if *FlagTcping {
		RunTcping()
		mux.Handle("/metrics", promhttp.Handler())
	}

	if *FlagMsg {
		MClient = NewMongoClient("mongodb://localhost:27017", "msglist", "msg")
		if MClient == nil {
			panic("new mongo client error")
		}
		mux.HandleFunc("/t", CreateMsg)
		mux.HandleFunc("/t/list", MsgList)
		mux.HandleFunc("/t/list/", MsgShow)
	}

	if *FlagTracer {
		mux.HandleFunc("/tracer", HandleTracer)
	}

	mux.HandleFunc("/ip", CFIPHandler)
	mux.HandleFunc("/memfile/", MemFileHandler)
	mux.HandleFunc("/vnstat", VnstatHandler)
	mux.HandleFunc("/upload", Uploader)

	mux.Handle("/", http.FileServer(http.Dir("./")))

	addr := ":" + strconv.FormatUint(uint64(*FlagPort), 10)
	log.Println("listen on:", addr)
	err := http.ListenAndServe(addr, &mux)
	if err != nil {
		log.Println("listen and serve error is:", err)
	}
}
