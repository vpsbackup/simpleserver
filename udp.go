package main

import (
	"net"
	"strconv"
	"log"
)

func ListenUDP(port int) {
	log.Println("udp listen on:", port)
	udpAddr, err := net.ResolveUDPAddr("udp", ":" + strconv.FormatInt(int64(port), 10))
	if err != nil {
		log.Println("parse udp addr+port error:", err)
		return
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Println("listen udp error:", udpAddr, err)
		return
	}

	for {
		var buf [512]byte
		n, addr, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			log.Println("read udp error:", err)
			continue
		}
		log.Println("udp client is:", addr)
		log.Println("we read udp:", n)
		conn.WriteToUDP(buf[:n], addr)
	}
}
