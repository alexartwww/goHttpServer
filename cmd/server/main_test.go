package main

import (
	"bufio"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestServe(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "index.html"), []byte("<h1>home</h1>"), 0644)
	os.Mkdir(filepath.Join(root, "docs"), 0755)
	os.WriteFile(filepath.Join(root, "docs", "a.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(root, "secret.txt"), []byte("no"), 0000)
	os.WriteFile(filepath.Join(filepath.Dir(root), "outside.txt"), []byte("leak"), 0644)
	setHome(root)

	tests := []struct {
		name       string
		request    string
		wantStatus string
		wantBody   string
	}{
		{"index", "GET / HTTP/1.1\r\n\r\n", "200 OK", "<h1>home</h1>"},
		{"file", "GET /docs/a.txt HTTP/1.1\r\n\r\n", "200 OK", "hello"},
		{"head has no body", "HEAD /docs/a.txt HTTP/1.1\r\n\r\n", "200 OK", ""},
		{"not found", "GET /nope HTTP/1.1\r\n\r\n", "404 Not Found", ""},
		{"traversal", "GET /../outside.txt HTTP/1.1\r\n\r\n", "404 Not Found", ""},
		{"encoded traversal", "GET /docs/..%2f..%2foutside.txt HTTP/1.1\r\n\r\n", "404 Not Found", ""},
		{"bad escape", "GET /%zz HTTP/1.1\r\n\r\n", "400 Bad Request", ""},
		{"post", "POST / HTTP/1.1\r\n\r\n", "405 Method Not Allowed", ""},
		{"options", "OPTIONS * HTTP/1.1\r\n\r\n", "204 No Content", ""},
		{"not modified", "GET /docs/a.txt HTTP/1.1\r\nIf-Modified-Since: Fri, 01 Jan 2100 00:00:00 GMT\r\n\r\n", "304 Not Modified", ""},
		{"bad version", "GET / HTTP/2.0\r\n\r\n", "505 HTTP Version Not Supported", ""},
	}
	if os.Geteuid() != 0 {
		tests = append(tests, struct {
			name, request, wantStatus, wantBody string
		}{"unreadable", "GET /secret.txt HTTP/1.1\r\n\r\n", "403 Forbidden", ""})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, headers, body := roundTrip(t, tt.request)
			if status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", status, tt.wantStatus)
			}
			if tt.wantBody != "" && body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
			if strings.HasPrefix(tt.request, "HEAD") && body != "" {
				t.Errorf("HEAD response has a body: %q", body)
			}
			if !strings.HasSuffix(headers["Date"], " GMT") {
				t.Errorf("Date = %q, want GMT", headers["Date"])
			}
		})
	}
}

func TestNotModifiedByETag(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello"), 0644)
	setHome(root)

	_, headers, _ := roundTrip(t, "GET /a.txt HTTP/1.1\r\n\r\n")
	etag := headers["ETag"]
	if etag == "" {
		t.Fatal("no ETag")
	}
	status, _, body := roundTrip(t, "GET /a.txt HTTP/1.1\r\nIf-None-Match: "+etag+"\r\n\r\n")
	if status != "304 Not Modified" || body != "" {
		t.Errorf("got %q with body %q, want 304 without body", status, body)
	}
}

func roundTrip(t *testing.T, request string) (status string, headers map[string]string, body string) {
	t.Helper()
	client, server := net.Pipe()
	defer client.Close()
	go handleClient(server, testHosts)
	go client.Write([]byte(request))
	return readResponse(t, bufio.NewReader(client), strings.HasPrefix(request, "HEAD"))
}

func readResponse(t *testing.T, reader *bufio.Reader, head bool) (status string, headers map[string]string, body string) {
	t.Helper()
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read status line: %v", err)
	}
	status = strings.TrimSpace(strings.SplitN(line, " ", 2)[1])
	headers = map[string]string{}
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read header: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, _ := strings.Cut(line, ": ")
		headers[name] = value
	}
	if head {
		return status, headers, ""
	}
	length, _ := strconv.Atoi(headers["Content-Length"])
	rest := make([]byte, length)
	if _, err := io.ReadFull(reader, rest); err != nil {
		t.Fatalf("read body: %v", err)
	}
	return status, headers, string(rest)
}

func TestKeepAlive(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("aaa"), 0644)
	os.WriteFile(filepath.Join(root, "b.txt"), []byte("bb"), 0644)
	setHome(root)

	client, server := net.Pipe()
	defer client.Close()
	finished := make(chan struct{})
	go func() {
		handleClient(server, testHosts)
		close(finished)
	}()
	reader := bufio.NewReader(client)

	// Two requests in one write: the second one is already in the buffer after the first.
	go client.Write([]byte("GET /a.txt HTTP/1.1\r\n\r\nHEAD /b.txt HTTP/1.1\r\n\r\n"))
	if status, headers, body := readResponse(t, reader, false); status != "200 OK" || body != "aaa" || headers["Connection"] != "keep-alive" {
		t.Fatalf("first: %q %q %q", status, headers["Connection"], body)
	}
	if status, headers, _ := readResponse(t, reader, true); status != "200 OK" || headers["Content-Length"] != "2" {
		t.Fatalf("second (pipelined): %q %v", status, headers)
	}

	go client.Write([]byte("GET /b.txt HTTP/1.1\r\nConnection: close\r\n\r\n"))
	if status, headers, body := readResponse(t, reader, false); status != "200 OK" || body != "bb" || headers["Connection"] != "close" {
		t.Fatalf("third: %q %q %q", status, headers["Connection"], body)
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("connection was not closed after Connection: close")
	}
}

func TestCloseAfter(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("aaa"), 0644)
	setHome(root)

	for name, request := range map[string]string{
		"http 1.0":       "GET /a.txt HTTP/1.0\r\n\r\n",
		"request body":   "POST /a.txt HTTP/1.1\r\nContent-Length: 3\r\n\r\nxyz",
		"parse error":    "GE(T / HTTP/1.1\r\n\r\n",
		"http 1.0 close": "GET /a.txt HTTP/1.0\r\nConnection: close\r\n\r\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, headers, _ := roundTrip(t, request); headers["Connection"] != "close" {
				t.Errorf("Connection = %q, want close", headers["Connection"])
			}
		})
	}
}

func TestLargeFile(t *testing.T) {
	root := t.TempDir()
	big := strings.Repeat("0123456789abcdef", (smallFileSize/16)*3+7)
	os.WriteFile(filepath.Join(root, "big.bin"), []byte(big), 0644)
	setHome(root)

	status, headers, body := roundTrip(t, "GET /big.bin HTTP/1.1\r\n\r\n")
	if status != "200 OK" || headers["Content-Length"] != strconv.Itoa(len(big)) || body != big {
		t.Fatalf("status %q, Content-Length %q, body %d bytes, want %d", status, headers["Content-Length"], len(body), len(big))
	}
}

var testHosts []*vhost

func setHome(root string) {
	vh, err := newVhost(ServerConfig{Home: root})
	if err != nil {
		panic(err)
	}
	testHosts = []*vhost{vh}
	fileCacheBytes.Store(0)
}

func TestCacheRefresh(t *testing.T) {
	root := t.TempDir()
	name := filepath.Join(root, "a.txt")
	os.WriteFile(name, []byte("old"), 0644)
	setHome(root)

	for i := 0; i < 2; i++ {
		if _, _, body := roundTrip(t, "GET /a.txt HTTP/1.1\r\n\r\n"); body != "old" {
			t.Fatalf("request %d: body %q", i, body)
		}
	}
	os.WriteFile(name, []byte("new!"), 0644)
	os.Chtimes(name, time.Now().Add(time.Minute), time.Now().Add(time.Minute))
	time.Sleep(cacheValid + 100*time.Millisecond)
	if _, headers, body := roundTrip(t, "GET /a.txt HTTP/1.1\r\n\r\n"); body != "new!" || headers["Content-Length"] != "4" {
		t.Fatalf("after change: body %q, Content-Length %q", body, headers["Content-Length"])
	}
}
