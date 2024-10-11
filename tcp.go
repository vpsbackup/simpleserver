package main

import (
	"log"
	"net"
	"strconv"
)

func ListenTCP(port int) {
	log.Println("tcp listen on:", port)
	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+strconv.FormatInt(int64(port), 10))
	if err != nil {
		log.Println("parse tcp addr error:", err)
		return
	}
	ls, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Println("listen tcp error:", tcpAddr, err)
		return
	}

	for {
		conn, err := ls.Accept()
		if err != nil {
			log.Println("tcp accept error:", err)
			continue
		}
		handleTcp(conn)
	}
}

func handleTcp(conn net.Conn) {
	defer conn.Close()

	for {
		buf := make([]byte, 100)
		n, err := conn.Read(buf)
		if err != nil {
			log.Println("read tcp error:", err)
			return
		}
		log.Println("read from:", conn.RemoteAddr(), n)
		_, err = conn.Write(buf[:n])
		if err != nil {
			log.Println("write tcp error:", err)
			return
		}
	}
}
