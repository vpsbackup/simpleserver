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
var FlagDomain = flag.String("d", "x.871116.xyz", "domain name in upload")
var FlagTcping = flag.Bool("t", false, "tcping")
var FlagMsg = flag.Bool("m", false, "message service")
var FlagTcping6 = flag.Bool("t6", true, "tcping IPv6")
var FlagV = flag.Bool("v", false, "version")
var FlagMock = flag.Bool("mock", false, "is mock")
var FlagTracer = flag.Bool("tr", false, "enable tracer")
var FlagV4Fn = flag.String("v4fn", "v4.txt", "ipv4 file name")
var FlagV6Fn = flag.String("v6fn", "v6.txt", "ipv6 file name")
var FlagEnableView = flag.Bool("ev", false, "enable view info")
var FlagMaxSingleFile = flag.Int64("msf", 100, "max single file MB")
var FlagMaxTotalFile = flag.Int64("mtf", 10, "max total file in GB")
var FlagPeers = flag.String("peer", "", "domains with comma seperated")
var FlagUdpPort = flag.Int("up", 10081, "udp listen port number")

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
	log.Println("Tracer:", *FlagTracer)
	log.Println("V4Fn:", *FlagV4Fn)
	log.Println("V6Fn:", *FlagV6Fn)
	log.Println("EnableView:", *FlagEnableView)
	log.Println("MaxSingleFileMB:", *FlagMaxSingleFile)
	log.Println("MaxTotalFileGB:", *FlagMaxTotalFile)
	log.Println("Peers:", *FlagPeers)
	log.Println("Udp Port:", *FlagUdpPort)

	if *FlagMaxSingleFile*1024*1024 > MaxHTTPPayload {
		MaxHTTPPayload = *FlagMaxSingleFile * 1024 * 1024
	}
	if *FlagMaxTotalFile*1024*1024*1024 > MaxTotalFileSize {
		MaxTotalFileSize = *FlagMaxTotalFile * 1024 * 1024 * 1024
	}

	if *FlagV {
		Version()
		return
	}

	if *FlagUdpPort != 0 {
		go ListenUDP(*FlagUdpPort)
	}

	InitPeers()

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
			return
		}
		mux.HandleFunc("/ip/", GlobalViewService.Handle)
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
	mux.HandleFunc("/vnstat.all", VnstatAllHandler)
	mux.HandleFunc("/api/", ApiHandler)
	mux.HandleFunc("/upload", Uploader)

	mux.Handle("/", http.FileServer(http.Dir("./")))

	addr := ":" + strconv.FormatUint(uint64(*FlagPort), 10)
	log.Println("listen on:", addr)
	err := http.ListenAndServe(addr, &mux)
	if err != nil {
		log.Println("listen and serve error is:", err)
	}
}
