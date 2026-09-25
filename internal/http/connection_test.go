package http

import "testing"

func TestKeepAlive(t *testing.T) {
	tests := []struct {
		raw  string
		want bool
	}{
		{"GET / HTTP/1.1\r\n\r\n", true},
		{"GET / HTTP/1.1\r\nConnection: close\r\n\r\n", false},
		{"GET / HTTP/1.1\r\nConnection: Upgrade, CLOSE\r\n\r\n", false},
		{"GET / HTTP/1.0\r\n\r\n", false},
		{"GET / HTTP/1.0\r\nConnection: Keep-Alive\r\n\r\n", true},
	}
	for _, tt := range tests {
		if got := parse(t, tt.raw).KeepAlive(); got != tt.want {
			t.Errorf("KeepAlive(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestHasBody(t *testing.T) {
	tests := []struct {
		raw  string
		want bool
	}{
		{"GET / HTTP/1.1\r\n\r\n", false},
		{"POST / HTTP/1.1\r\nContent-Length: 0\r\n\r\n", false},
		{"POST / HTTP/1.1\r\nContent-Length: 5\r\n\r\n", true},
		{"POST / HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n", true},
	}
	for _, tt := range tests {
		if got := parse(t, tt.raw).HasBody(); got != tt.want {
			t.Errorf("HasBody(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}
