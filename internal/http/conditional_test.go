package http

import (
	"testing"
	"time"
)

func TestNotModified(t *testing.T) {
	modified := time.Date(2026, 9, 24, 10, 0, 0, 500, time.UTC)
	etag := `"66f28e20-1a4"`
	tests := []struct {
		name    string
		headers string
		want    bool
	}{
		{"no conditions", "", false},
		{"etag match", "If-None-Match: \"66f28e20-1a4\"\r\n", true},
		{"etag in list", "If-None-Match: \"x\", W/\"66f28e20-1a4\"\r\n", true},
		{"etag star", "If-None-Match: *\r\n", true},
		{"etag mismatch", "If-None-Match: \"other\"\r\n", false},
		{"same second", "If-Modified-Since: Thu, 24 Sep 2026 10:00:00 GMT\r\n", true},
		{"later", "If-Modified-Since: Thu, 24 Sep 2026 11:00:00 GMT\r\n", true},
		{"earlier", "If-Modified-Since: Thu, 24 Sep 2026 09:59:59 GMT\r\n", false},
		{"rfc 850", "If-Modified-Since: Thursday, 24-Sep-26 10:00:00 GMT\r\n", true},
		{"asctime", "If-Modified-Since: Thu Sep 24 10:00:00 2026\r\n", true},
		{"bad date", "If-Modified-Since: yesterday\r\n", false},
		{"etag wins over date", "If-None-Match: \"other\"\r\nIf-Modified-Since: Thu, 24 Sep 2026 11:00:00 GMT\r\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parse(t, "GET / HTTP/1.1\r\n"+tt.headers+"\r\n")
			if got := NotModified(p, etag, modified); got != tt.want {
				t.Errorf("NotModified = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	moscow := time.FixedZone("MSK", 3*60*60)
	got := FormatTime(time.Date(2026, 9, 24, 13, 0, 0, 0, moscow))
	if got != "Thu, 24 Sep 2026 10:00:00 GMT" {
		t.Errorf("FormatTime = %q", got)
	}
}
