package main

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestHandleTCPEcho(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	go handleTCPConn(server, 2*time.Second)

	msg := []byte("hello echo")
	if _, err := client.Write(msg); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := client.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(msg))
	if _, err := io.ReadFull(client, got); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(got) != string(msg) {
		t.Fatalf("echo = %q, want %q", got, msg)
	}
}

// 回归测试：一个不说话的空闲连接不能阻塞 accept 循环，后连上的客户端照常 echo。
func TestServeTCPIdleConnDoesNotBlockOthers(t *testing.T) {
	ls, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer ls.Close()
	go ServeTCP(ls)

	// 第一个连接只建立、不发数据，模拟慢客户端。
	idle, err := net.Dial("tcp", ls.Addr().String())
	if err != nil {
		t.Fatalf("dial idle: %v", err)
	}
	defer idle.Close()

	// 第二个连接必须能在空闲连接超时之前拿到 echo。
	echo, err := net.Dial("tcp", ls.Addr().String())
	if err != nil {
		t.Fatalf("dial echo: %v", err)
	}
	defer echo.Close()
	if err := echo.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := echo.Write([]byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := make([]byte, 4)
	if _, err := io.ReadFull(echo, got); err != nil {
		t.Fatalf("second client blocked by idle conn: %v", err)
	}
	if string(got) != "ping" {
		t.Fatalf("echo = %q, want ping", got)
	}
}

// 空闲连接到期后由服务端关闭，不会永久占着 goroutine。
func TestHandleTCPIdleTimeout(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	done := make(chan struct{})
	go func() {
		handleTCPConn(server, 100*time.Millisecond)
		close(done)
	}()

	if err := client.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1)
	if _, err := client.Read(buf); err == nil {
		t.Fatal("expected read error after server closed idle conn")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleTcp did not return after idle timeout")
	}
}

// 连接数达到上限时应快速丢弃新连接，而不是无限增长 goroutine。
func TestServeTCPDropsWhenBusy(t *testing.T) {
	// 占满全部 slot
	for i := 0; i < tcpEchoMaxConns; i++ {
		tcpEchoSlots <- struct{}{}
	}
	t.Cleanup(func() {
		for i := 0; i < tcpEchoMaxConns; i++ {
			<-tcpEchoSlots
		}
	})

	ls, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer ls.Close()
	go ServeTCP(ls)

	conn, err := net.Dial("tcp", ls.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	// 服务端直接 Close，客户端读到 EOF 而不是 echo。
	if _, err := conn.Write([]byte("hi")); err == nil {
		buf := make([]byte, 2)
		if _, err := io.ReadFull(conn, buf); err == nil {
			t.Fatalf("busy server echoed %q, want closed conn", buf)
		}
	}
}
