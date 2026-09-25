package main

import (
	"bufio"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

// System events go to stderr through slog, the access log goes to stdout.
func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
}

var logTime atomic.Pointer[string]

var accessLog = bufio.NewWriterSize(os.Stdout, 64<<10)
var accessLogMu sync.Mutex

type accessEntry struct {
	remote string
	host   []byte
	method []byte
	uri    []byte
	status uint16
}

func logRequest(line []byte, vh *vhost, e *accessEntry) []byte {
	if vh.logJSON {
		line = appendAccessJSON(line[:0], e)
	} else {
		line = appendAccessText(line[:0], e)
	}
	accessLogMu.Lock()
	accessLog.Write(line)
	accessLogMu.Unlock()
	return line
}

func appendAccessText(line []byte, e *accessEntry) []byte {
	line = append(line, *logTime.Load()...)
	line = append(line, ' ')
	line = append(line, e.remote...)
	line = append(line, ' ')
	line = strconv.AppendUint(line, uint64(e.status), 10)
	line = append(line, ' ')
	line = append(line, e.method...)
	line = append(line, ' ')
	line = append(line, e.uri...)
	return append(line, '\n')
}

// Built by hand instead of encoding/json: no reflection and no allocations per request.
func appendAccessJSON(line []byte, e *accessEntry) []byte {
	line = append(line, `{"time":"`...)
	line = append(line, *logTime.Load()...)
	line = append(line, `","remote":`...)
	line = appendJSONString(line, e.remote)
	line = append(line, `,"host":`...)
	line = appendJSONString(line, e.host)
	line = append(line, `,"method":`...)
	line = appendJSONString(line, e.method)
	line = append(line, `,"uri":`...)
	line = appendJSONString(line, e.uri)
	line = append(line, `,"status":`...)
	line = strconv.AppendUint(line, uint64(e.status), 10)
	return append(line, "}\n"...)
}

const hexDigits = "0123456789abcdef"

// Header values come from the client: quotes, control characters and non-ASCII bytes are escaped
// so a request cannot break the JSON line or inject a fake one.
func appendJSONString[T string | []byte](line []byte, s T) []byte {
	line = append(line, '"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"' || c == '\\':
			line = append(line, '\\', c)
		case c < 0x20 || c >= 0x7f:
			line = append(line, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
		default:
			line = append(line, c)
		}
	}
	return append(line, '"')
}
