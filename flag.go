package main

import (
	"flag"
	"log"
)

var FlagBehindNginx = flag.Bool("bn", true, "behind nginx")
var FlagPort = flag.Int("p", 10080, "port number")
var FlagDomain = flag.String("d", "x.871116.xyz", "domain name in upload")
var FlagTcping = flag.Bool("t", false, "tcping")
var FlagMsg = flag.Bool("m", false, "message service")
var FlagTcping6 = flag.Bool("t6", false, "tcping IPv6")
var FlagV = flag.Bool("v", false, "version")
var FlagMock = flag.Bool("mock", false, "is mock")
var FlagTracer = flag.Bool("tr", false, "enable tracer")
var FlagV4Fn = flag.String("v4fn", "v4.txt", "ipv4 file name")
var FlagV6Fn = flag.String("v6fn", "v6.txt", "ipv6 file name")
var FlagEnableView = flag.Bool("ev", false, "enable view info")
var FlagMaxSingleFile = flag.Int64("msf", 100, "max single file MB")
var FlagMaxTotalFile = flag.Int64("mtf", 10, "max total file in GB")
var FlagUdpPort = flag.Int("up", 10081, "udp listen port number")
var FlagTcpPort = flag.Int("tp", 10082, "tcp listen port number")
var FlagQuicAddr = flag.String("h3a", ":443", "quic listen addr")
var FlagQuicCertPath = flag.String("h3c", "/root/sh/cert.pem", "tls cert file")
var FlagQuicKeyPath = flag.String("h3p", "/root/sh/priv.pem", "tls cert file")
var FlagQuicOnly = flag.Bool("h3o", false, "quic only")
var FlagUseQuic = flag.Bool("h3e", true, "use quic")

func ParseFlag() {
	flag.Parse()

	if *FlagMaxSingleFile*1024*1024 > MaxHTTPPayload {
		MaxHTTPPayload = *FlagMaxSingleFile * 1024 * 1024
	}
	if *FlagMaxTotalFile*1024*1024*1024 > MaxTotalFileSize {
		MaxTotalFileSize = *FlagMaxTotalFile * 1024 * 1024 * 1024
	}

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
	log.Println("Udp Port:", *FlagUdpPort)
	log.Println("Tcp Port:", *FlagTcpPort)
	log.Println("Quic Addr:", *FlagQuicAddr)
	log.Println("Quic Cert Path:", *FlagQuicCertPath)
	log.Println("Quic Key Path:", *FlagQuicKeyPath)
	log.Println("Quic Only:", *FlagQuicOnly)
	log.Println("Use Quic:", *FlagUseQuic)
}
