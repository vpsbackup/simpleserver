package main

import (
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"time"
)

const (
	tcpEchoBufSize  = 4096
	tcpEchoMaxConns = 512
	// tcpEchoIdleTimeout 是单条 echo 连接允许的最长空闲时间。
	tcpEchoIdleTimeout = 60 * time.Second
)

// tcpEchoSlots 限制同时服务的连接数，避免大量连接把 goroutine 和内存打满。
var tcpEchoSlots = make(chan struct{}, tcpEchoMaxConns)

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
	ServeTCP(ls)
}

// ServeTCP echoes every byte back. 每个连接单独一个 goroutine，
// 慢连接或空闲连接不会阻塞 accept 循环。
func ServeTCP(ls *net.TCPListener) {
	for {
		conn, err := ls.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Println("tcp listener closed:", ls.Addr())
				return
			}
			log.Println("tcp accept error:", err)
			continue
		}
		select {
		case tcpEchoSlots <- struct{}{}:
			go func(c net.Conn) {
				defer func() { <-tcpEchoSlots }()
				handleTcp(c)
			}(conn)
		default:
			log.Println("tcp echo is busy, dropping conn from:", conn.RemoteAddr())
			conn.Close()
		}
	}
}

func handleTcp(conn net.Conn) {
	handleTCPConn(conn, tcpEchoIdleTimeout)
}

// handleTCPConn 是 echo 的实际实现；idle 单独传参，方便测试用短超时。
func handleTCPConn(conn net.Conn, idle time.Duration) {
	defer conn.Close()
	buf := make([]byte, tcpEchoBufSize)
	for {
		// 空闲连接不会被永久占用。
		if err := conn.SetReadDeadline(time.Now().Add(idle)); err != nil {
			return
		}
		n, err := conn.Read(buf)
		if err != nil {
			if !errors.Is(err, io.EOF) && !isNetTimeout(err) {
				log.Println("read tcp error:", err, "from:", conn.RemoteAddr())
			}
			return
		}
		if err := conn.SetWriteDeadline(time.Now().Add(idle)); err != nil {
			return
		}
		if _, err := conn.Write(buf[:n]); err != nil {
			log.Println("write tcp error:", err, "to:", conn.RemoteAddr())
			return
		}
	}
}

func isNetTimeout(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}
