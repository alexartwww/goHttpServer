package main

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "config.yml")
	os.WriteFile(name, []byte(body), 0644)
	return name
}

func TestLoadConfig(t *testing.T) {
	home := t.TempDir()
	loaded, listeners, err := loadConfig(writeConfig(t, `
serverName: test/1.0
readTimeout: 2s
maxRequests: 10
servers:
  - listen: 127.0.0.1:8080
    home: `+home+`
  - listen: 127.0.0.1:8080
    host: Example.COM
    home: `+home+`/
    indexes: [main.html]
    logFormat: text
  - listen: 127.0.0.1:8081
    home: `+home))
	if err != nil {
		t.Fatal(err)
	}
	want := Settings{ServerName: "test/1.0", ReadTimeout: Duration(2 * time.Second), WriteTimeout: defaultSettings.WriteTimeout, MaxRequests: 10}
	if loaded != want {
		t.Errorf("settings = %+v, want %+v", loaded, want)
	}
	if len(listeners) != 2 || len(listeners["127.0.0.1:8080"]) != 2 {
		t.Fatalf("listeners = %v", listeners)
	}
	def, example := listeners["127.0.0.1:8080"][0], listeners["127.0.0.1:8080"][1]
	if def.host != "default" || !def.logJSON || strings.Join(def.indexes, ",") != ",index.html,index.htm" {
		t.Errorf("default = %+v", def)
	}
	if example.host != "example.com" || example.logJSON || example.home != home || strings.Join(example.indexes, ",") != ",main.html" {
		t.Errorf("example = %+v", example)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	home := t.TempDir()
	for name, body := range map[string]string{
		"no servers":     "servers: []",
		"no home":        "servers:\n  - listen: :80",
		"missing home":   "servers:\n  - home: " + home + "/nope",
		"bad log format": "servers:\n  - home: " + home + "\n    logFormat: xml",
		"duplicate host": "servers:\n  - home: " + home + "\n  - home: " + home,
		"bad yaml":       "servers: [",
		"header in name": "serverName: \"x\\r\\nSet-Cookie: a=b\"\nservers:\n  - home: " + home,
		"bad timeout":    "readTimeout: 5\nservers:\n  - home: " + home,
		"negative":       "writeTimeout: -1s\nservers:\n  - home: " + home,
		"zero requests":  "maxRequests: 0\nservers:\n  - home: " + home,
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := loadConfig(writeConfig(t, body)); err == nil {
				t.Error("no error")
			}
		})
	}
}

func TestPickHost(t *testing.T) {
	hosts := []*vhost{}
	for _, h := range []string{"a.test", "default", "[::1]"} {
		vh, _ := newVhost(ServerConfig{Host: h, Home: t.TempDir()})
		hosts = append(hosts, vh)
	}
	for host, want := range map[string]string{
		"a.test":      "a.test",
		"A.TEST:8080": "a.test",
		"b.test":      "default",
		"":            "default",
		"[::1]:80":    "[::1]",
	} {
		if got := pickHost(hosts, []byte(host)).host; got != want {
			t.Errorf("pickHost(%q) = %q, want %q", host, got, want)
		}
	}
}

func TestVirtualHosts(t *testing.T) {
	var hosts []*vhost
	for _, h := range []string{"default", "b.test"} {
		root := t.TempDir()
		os.WriteFile(filepath.Join(root, "index.html"), []byte(h), 0644)
		vh, _ := newVhost(ServerConfig{Host: h, Home: root})
		hosts = append(hosts, vh)
	}
	for host, want := range map[string]string{"b.test": "b.test", "other.test": "default"} {
		client, server := net.Pipe()
		go handleClient(server, hosts)
		go client.Write([]byte("GET / HTTP/1.1\r\nHost: " + host + "\r\nConnection: close\r\n\r\n"))
		if _, _, body := readResponse(t, bufio.NewReader(client), false); body != want {
			t.Errorf("Host %s: body %q, want %q", host, body, want)
		}
		client.Close()
	}
}
