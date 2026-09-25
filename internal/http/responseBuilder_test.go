package http

import "testing"

func TestResponseBuild(t *testing.T) {
	response := Response{Protocol: "HTTP", Version: "1.1", Code: 304, Status: "Not Modified",
		Headers: []Header{{Name: "ETag", Value: `"x"`}}}
	if got := string(response.Build()); got != "HTTP/1.1 304 Not Modified\r\nETag: \"x\"\r\n\r\n" {
		t.Errorf("without body: %q", got)
	}
	response.Body = []byte("hi")
	if got := string(response.Build()); got != "HTTP/1.1 304 Not Modified\r\nETag: \"x\"\r\n\r\nhi" {
		t.Errorf("with body: %q", got)
	}
}
