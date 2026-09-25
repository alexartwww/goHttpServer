//go:build !linux

package main

import "net"

func setCork(conn net.Conn, on bool) {}
