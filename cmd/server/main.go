package main

import (
	"flag"
	"io"
	"log/slog"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alexartwww/goHttpServer/internal/file"
	"github.com/alexartwww/goHttpServer/internal/http"
)

var allowedMethods = []string{"GET", "HEAD", "OPTIONS"}

// Files up to this size are cached in memory and go out in one write with the head, bigger ones via sendfile.
const smallFileSize = 64 << 10

var readBufPool = sync.Pool{New: func() any { b := make([]byte, 0, 4096); return &b }}
var writeBufPool = sync.Pool{New: func() any { b := make([]byte, 0, 4096); return &b }}

var dateHeader atomic.Pointer[string]

func init() {
	updateDate()
	go func() {
		for range time.Tick(time.Second) {
			updateDate()
			accessLogMu.Lock()
			accessLog.Flush()
			accessLogMu.Unlock()
		}
	}()
}

func updateDate() {
	now := time.Now()
	date := http.FormatTime(now)
	dateHeader.Store(&date)
	logDate := now.Format(time.RFC3339)
	logTime.Store(&logDate)
}

func main() {
	configFile := flag.String("config", "", "config file, searched in config/ and /etc when empty")
	flag.Parse()

	loaded, listeners, err := loadConfig(*configFile)
	if err != nil {
		slog.Error("config",
			"error", err,
		)
		os.Exit(1)
	}
	settings = loaded
	slog.Info("settings",
		"server_name", settings.ServerName,
		"read_timeout", settings.ReadTimeout.String(),
		"write_timeout", settings.WriteTimeout.String(),
		"max_requests", settings.MaxRequests,
	)

	failed := make(chan error)
	for address, hosts := range listeners {
		listener, err := net.Listen("tcp", address)
		if err != nil {
			slog.Error("listen", "address", address, "error", err)
			os.Exit(1)
		}
		for _, vh := range hosts {
			slog.Info("listen", "address", address, "host", vh.host, "home", vh.home, "json_log", vh.logJSON)
		}
		go func() { failed <- serveListener(listener, hosts) }()
	}
	slog.Error("accept", "error", <-failed)
	os.Exit(1)
}

func serveListener(listener net.Listener, hosts []*vhost) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return err
		}
		go handleClient(conn, hosts)
	}
}

type reply struct {
	response http.Response
	file     *os.File
	size     int64
}

func handleClient(conn net.Conn, hosts []*vhost) {
	defer conn.Close()

	readBuf := readBufPool.Get().(*[]byte)
	writeBuf := writeBufPool.Get().(*[]byte)
	buff, out := (*readBuf)[:0], (*writeBuf)[:0]
	defer func() {
		*readBuf, *writeBuf = buff, out
		readBufPool.Put(readBuf)
		writeBufPool.Put(writeBuf)
	}()

	entry := accessEntry{remote: conn.RemoteAddr().String()}
	logLine := make([]byte, 0, 256)

	var request http.Parser
	for served := 1; ; served++ {
		// Idle keep-alive time and a slow head (Slowloris) are both limited by this deadline.
		conn.SetReadDeadline(time.Now().Add(time.Duration(settings.ReadTimeout)))
		request.Reset()
		// Pipelined bytes of this request may be in the buffer already.
		done, parseErr := request.Feed(buff)
		for !done && parseErr == nil {
			buff = slices.Grow(buff, 1024)
			readLen, err := conn.Read(buff[len(buff):cap(buff)])
			buff = buff[:len(buff)+readLen]
			done, parseErr = request.Feed(buff)
			if err != nil && !done && parseErr == nil {
				if err != io.EOF && !isTimeout(err) {
					slog.Warn("read", "remote", entry.remote, "error", err)
				}
				return
			}
		}

		currentTime := time.Now()
		if parseErr != nil {
			response := errorResponse(currentTime, parseErr.(*http.Error))
			response.Headers = append(response.Headers, http.Header{Name: "Connection", Value: "close"})
			entry.host, entry.method, entry.uri, entry.status = nil, nil, nil, response.Code
			logRequest(logLine, pickHost(hosts, nil), &entry)
			conn.Write(response.Build())
			return
		}

		entry.host = request.Header("Host")
		vh := pickHost(hosts, entry.host)
		r := serve(&request, vh, currentTime)
		if string(request.Method()) == "HEAD" {
			r.response.Body = nil
		}
		// The body is not read, so after a request with one the connection cannot be reused.
		keepAlive := request.KeepAlive() && !request.HasBody() && served < settings.MaxRequests
		if keepAlive {
			r.response.Headers = append(r.response.Headers, http.Header{Name: "Connection", Value: "keep-alive"})
		} else {
			r.response.Headers = append(r.response.Headers, http.Header{Name: "Connection", Value: "close"})
		}
		entry.method, entry.uri, entry.status = request.Method(), request.Target(), r.response.Code
		logLine = logRequest(logLine, vh, &entry)
		conn.SetWriteDeadline(time.Now().Add(time.Duration(settings.WriteTimeout)))
		var err error
		out, err = writeReply(conn, r.response.AppendHead(out[:0]), &r)
		if err != nil || !keepAlive {
			return
		}

		buff = buff[:copy(buff, buff[request.HeadLen():])]
	}
}

func writeReply(conn net.Conn, out []byte, r *reply) ([]byte, error) {
	if r.file == nil {
		out = append(out, r.response.Body...)
		_, err := conn.Write(out)
		return out, err
	}
	defer r.file.Close()
	setCork(conn, true)
	defer setCork(conn, false)
	if _, err := conn.Write(out); err != nil {
		return out, err
	}
	// On a TCP connection this is sendfile: the file goes to the socket without copying through user space.
	_, err := io.Copy(conn, r.file)
	return out, err
}

func isTimeout(err error) bool {
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

func serve(request *http.Parser, vh *vhost, currentTime time.Time) reply {
	switch string(request.Method()) {
	case "GET", "HEAD":
	case "OPTIONS":
		response := newResponse(204, "No Content")
		response.Headers = append(response.Headers, http.Header{Name: "Allow", Value: strings.Join(allowedMethods, ", ")})
		return reply{response: response}
	default:
		response := errorResponse(currentTime, &http.Error{Status: 405, Reason: "Method Not Allowed"})
		response.Headers = append(response.Headers, http.Header{Name: "Allow", Value: strings.Join(allowedMethods, ", ")})
		return reply{response: response}
	}

	// Decoded and cleaned: "..", "%2e%2e" and the like cannot leave the document root.
	cleanPath, pathErr := http.CleanPath(make([]byte, 0, 256), request.Path())
	if pathErr != nil {
		return reply{response: errorResponse(currentTime, pathErr.(*http.Error))}
	}
	if entry := vh.cache.get(cleanPath, currentTime); entry != nil {
		return entryReply(request, entry.etag, entry.modified, entry.headers, entry.body)
	}
	path := string(cleanPath)

	reader := file.File{}
	for _, index := range vh.indexes {
		fileLocation := vh.home + path
		if fileLocation[len(fileLocation)-1:] != "/" && index != "" {
			fileLocation = fileLocation + "/"
		}
		fileLocation = fileLocation + index
		reader.Info(fileLocation)
		if !reader.Exist || reader.Directory {
			continue
		}
		f, err := reader.Open()
		if err != nil {
			if os.IsPermission(err) {
				return reply{response: errorResponse(currentTime, &http.Error{Status: 403, Reason: "Forbidden"})}
			}
			continue
		}
		return fileReply(request, vh, &reader, f, path, currentTime)
	}
	return reply{response: errorResponse(currentTime, &http.Error{Status: 404, Reason: "Not Found"})}
}

func fileReply(request *http.Parser, vh *vhost, reader *file.File, f *os.File, path string, currentTime time.Time) reply {
	headers := fileHeaders(reader)
	send := string(request.Method()) != "HEAD" && !http.NotModified(request, reader.ETag, reader.DateTime)
	if !send || reader.Size > smallFileSize {
		r := entryReply(request, reader.ETag, reader.DateTime, headers, nil)
		if send {
			r.file, r.size = f, reader.Size
		} else {
			f.Close()
		}
		return r
	}

	body := make([]byte, reader.Size)
	_, err := io.ReadFull(f, body)
	f.Close()
	if err != nil {
		return reply{response: errorResponse(currentTime, &http.Error{Status: 500, Reason: "Internal Server Error"})}
	}
	vh.cache.put(path, reader, body, currentTime)
	return entryReply(request, reader.ETag, reader.DateTime, headers, body)
}

func entryReply(request *http.Parser, etag string, modified time.Time, headers []http.Header, body []byte) reply {
	response := newResponse(200, "OK")
	if http.NotModified(request, etag, modified) {
		response.Code = 304
		response.Status = "Not Modified"
		response.Headers = append(response.Headers, validators(headers)...)
		return reply{response: response}
	}
	response.Headers = append(response.Headers, headers...)
	if string(request.Method()) != "HEAD" {
		response.Body = body
	}
	return reply{response: response}
}

func newResponse(code uint16, status string) http.Response {
	headers := make([]http.Header, 2, 8)
	headers[0] = http.Header{Name: "Server", Value: settings.ServerName}
	headers[1] = http.Header{Name: "Date", Value: *dateHeader.Load()}
	return http.Response{
		Protocol: "HTTP",
		Version:  "1.1",
		Code:     code,
		Status:   status,
		Headers:  headers,
	}
}

func errorResponse(currentTime time.Time, err *http.Error) http.Response {
	response := newResponse(uint16(err.Status), err.Reason)
	text := strconv.Itoa(err.Status) + " " + err.Reason
	response.Body = []byte("<html><head><title>" + text + "</title></head><body><h1 style=\"text-align: center;\">" + text + "</h1><hr><p style=\"text-align: center;\">" + settings.ServerName + " " + http.FormatTime(currentTime) + "</p></body></html>")
	response.Headers = append(response.Headers,
		http.Header{Name: "Content-Type", Value: "text/html; charset=utf-8"},
		http.Header{Name: "Content-Length", Value: strconv.Itoa(len(response.Body))},
	)
	return response
}
