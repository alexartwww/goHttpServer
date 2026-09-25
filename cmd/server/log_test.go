package main

import (
	"encoding/json"
	"testing"
)

func TestAccessJSON(t *testing.T) {
	updateDate()
	e := accessEntry{
		remote: "127.0.0.1:5555",
		host:   []byte("evil\"}\n{\"status\":200\x01\xff"),
		method: []byte("GET"),
		uri:    []byte(`/a\b?c="d"`),
		status: 404,
	}
	line := appendAccessJSON(nil, &e)
	if line[len(line)-1] != '\n' {
		t.Fatalf("no newline: %q", line)
	}
	var got map[string]any
	if err := json.Unmarshal(line, &got); err != nil {
		t.Fatalf("invalid JSON %q: %v", line, err)
	}
	if got["host"] != "evil\"}\n{\"status\":200\x01ÿ" || got["uri"] != `/a\b?c="d"` || got["status"] != 404.0 || got["method"] != "GET" {
		t.Errorf("got %v", got)
	}
}

func TestAccessLogAllocs(t *testing.T) {
	updateDate()
	e := accessEntry{remote: "127.0.0.1:5555", host: []byte("a.test"), method: []byte("GET"), uri: []byte("/index.html"), status: 200}
	line := make([]byte, 0, 256)
	if n := testing.AllocsPerRun(100, func() { line = appendAccessJSON(line[:0], &e) }); n != 0 {
		t.Errorf("allocs = %v", n)
	}
}
