package main

import (
	"net"
	"syscall"
)

// Like nginx tcp_nopush: while corked, the head and the start of the file leave in full packets.
func setCork(conn net.Conn, on bool) {
	tcp, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}
	raw, err := tcp.SyscallConn()
	if err != nil {
		return
	}
	value := 0
	if on {
		value = 1
	}
	raw.Control(func(fd uintptr) {
		syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_CORK, value)
	})
}
