package main

import (
	"net"
	"testing"
	"time"
)

// #21: udp echo was previously untested; udpEcho must reply every packet back
// to the sender and exit when the socket is closed (net.ErrClosed).
func TestUdpEchoRoundTrip(t *testing.T) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	go udpEcho(conn)

	local := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	client, err := net.ListenUDP("udp", local)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })

	for _, msg := range []string{"hello", "second packet"} {
		if _, err := client.WriteToUDP([]byte(msg), conn.LocalAddr().(*net.UDPAddr)); err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 512)
		if err := client.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatal(err)
		}
		n, _, err := client.ReadFromUDP(buf)
		if err != nil {
			t.Fatalf("read echo: %v", err)
		}
		if string(buf[:n]) != msg {
			t.Fatalf("echo = %q, want %q", buf[:n], msg)
		}
	}
}

func TestUdpEchoExitsOnClose(t *testing.T) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		udpEcho(conn)
		close(done)
	}()
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("udpEcho did not exit after close")
	}
}
