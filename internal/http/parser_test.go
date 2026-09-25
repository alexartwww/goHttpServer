package http

import (
	"bytes"
	"testing"
)

var chromeRequest = []byte("GET /radio/listen/?a=1&b=2&FFSDF=asdasd HTTP/1.1\r\n" +
	"Host: artem-aleksashkin\r\n" +
	"Connection: keep-alive\r\n" +
	"Cache-Control: max-age=0\r\n" +
	"DNT: 1\r\n" +
	"Upgrade-Insecure-Requests: 1\r\n" +
	"User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.182 Safari/537.36\r\n" +
	"Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9\r\n" +
	"Referer: http://artem-aleksashkin/\r\n" +
	"Accept-Encoding: gzip, deflate\r\n" +
	"Accept-Language: ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7\r\n" +
	"Cookie: id=ca2a568d-cdf0-4658-b98c-320a9f1b0eb6; geography=1; timezone=Europe%2FMoscow; language=ru; language-data=ru%2Cen; currency=rub; user=c4ca4238a0b923820dcc509a6f75849b\r\n" +
	"\r\n")

func parse(t *testing.T, raw string) *Parser {
	t.Helper()
	var p Parser
	done, err := p.Feed([]byte(raw))
	if err != nil {
		t.Fatalf("Feed(%q) error: %v", raw, err)
	}
	if !done {
		t.Fatalf("Feed(%q) is not done", raw)
	}
	return &p
}

func TestParserRequestLine(t *testing.T) {
	p := parse(t, "GET /a/b.html?x=1&y=2 HTTP/1.1\r\n\r\n")
	check(t, "method", p.Method(), "GET")
	check(t, "target", p.Target(), "/a/b.html?x=1&y=2")
	check(t, "path", p.Path(), "/a/b.html")
	check(t, "query", p.Query(), "x=1&y=2")
	if major, minor := p.Version(); major != 1 || minor != 1 {
		t.Errorf("version = %d.%d, want 1.1", major, minor)
	}
	if p.HeadLen() != len("GET /a/b.html?x=1&y=2 HTTP/1.1\r\n\r\n") {
		t.Errorf("HeadLen = %d", p.HeadLen())
	}
}

func TestParserHeaders(t *testing.T) {
	p := parse(t, "GET / HTTP/1.0\r\nHost: example.com\r\nX-Spaces: \t a  b \t\r\nEmpty:\r\nx-lower: v\r\n\r\n")
	if p.NumHeaders() != 4 {
		t.Fatalf("NumHeaders = %d, want 4", p.NumHeaders())
	}
	check(t, "Host", p.Header("host"), "example.com")
	check(t, "X-Spaces", p.Header("X-SPACES"), "a  b")
	check(t, "Empty", p.Header("Empty"), "")
	check(t, "x-lower", p.Header("X-Lower"), "v")
	check(t, "name 0", p.HeaderName(0), "Host")
	if p.Header("Missing") != nil {
		t.Errorf("Missing header is not nil")
	}
}

func TestParserBareLF(t *testing.T) {
	// RFC 9112, 2.2: a recipient may accept a bare LF as a line terminator.
	p := parse(t, "GET /x HTTP/1.1\nHost: h\n\nbody")
	check(t, "Host", p.Header("Host"), "h")
	if got := len("GET /x HTTP/1.1\nHost: h\n\n"); p.HeadLen() != got {
		t.Errorf("HeadLen = %d, want %d", p.HeadLen(), got)
	}
}

func TestParserLeadingEmptyLines(t *testing.T) {
	p := parse(t, "\r\n\r\nGET / HTTP/1.1\r\n\r\n")
	check(t, "method", p.Method(), "GET")
}

func TestParserAsterisk(t *testing.T) {
	p := parse(t, "OPTIONS * HTTP/1.1\r\n\r\n")
	check(t, "path", p.Path(), "*")
}

func TestParserBodyIsNotConsumed(t *testing.T) {
	raw := "POST /form HTTP/1.1\r\nContent-Length: 4\r\n\r\nabcd"
	p := parse(t, raw)
	check(t, "body", []byte(raw[p.HeadLen():]), "abcd")
}

func TestParserQueryAndCookies(t *testing.T) {
	var p Parser
	if done, err := p.Feed(chromeRequest); !done || err != nil {
		t.Fatalf("Feed = %v, %v", done, err)
	}

	var query []string
	p.VisitQuery(func(name, value []byte) bool {
		query = append(query, string(name)+"="+string(value))
		return true
	})
	want := []string{"a=1", "b=2", "FFSDF=asdasd"}
	if !equalStrings(query, want) {
		t.Errorf("query = %q, want %q", query, want)
	}

	var cookies []string
	p.VisitCookies(func(name, value []byte) bool {
		cookies = append(cookies, string(name))
		return len(cookies) < 3
	})
	want = []string{"id", "geography", "timezone"}
	if !equalStrings(cookies, want) {
		t.Errorf("cookies = %q, want %q (stopped after 3)", cookies, want)
	}
}

func TestParserByteByByte(t *testing.T) {
	var whole, split Parser
	if done, err := whole.Feed(chromeRequest); !done || err != nil {
		t.Fatalf("whole Feed = %v, %v", done, err)
	}
	var done bool
	var err error
	for i := 1; i <= len(chromeRequest); i++ {
		done, err = split.Feed(chromeRequest[:i])
		if err != nil {
			t.Fatalf("split Feed at %d: %v", i, err)
		}
		if done != (i == len(chromeRequest)) {
			t.Fatalf("split Feed at %d: done = %v", i, done)
		}
	}
	assertSameParse(t, &whole, &split)
}

func TestParserErrors(t *testing.T) {
	long := string(bytes.Repeat([]byte("a"), MaxHeadSize))
	var manyHeaders string
	for i := 0; i <= MaxHeaders; i++ {
		manyHeaders += "X: y\r\n"
	}
	tests := []struct {
		name string
		raw  string
		want *Error
	}{
		{"no target", "GET HTTP/1.1\r\n\r\n", ErrBadRequest},
		{"space in method", "G ET / HTTP/1.1\r\n\r\n", ErrBadRequest},
		{"bad method byte", "GE(T / HTTP/1.1\r\n\r\n", ErrBadRequest},
		{"absolute form", "GET http://h/ HTTP/1.1\r\n\r\n", ErrBadRequest},
		{"control byte in path", "GET /a\x01 HTTP/1.1\r\n\r\n", ErrBadRequest},
		{"not http", "GET / FTP/1.1\r\n\r\n", ErrBadRequest},
		{"lowercase http", "GET / http/1.1\r\n\r\n", ErrBadRequest},
		{"http 2", "GET / HTTP/2.0\r\n\r\n", ErrVersion},
		{"http 1.2", "GET / HTTP/1.2\r\n\r\n", ErrVersion},
		{"junk after version", "GET / HTTP/1.1x\r\n\r\n", ErrBadRequest},
		{"CR without LF", "GET / HTTP/1.1\rX\r\n\r\n", ErrBadRequest},
		{"header without colon", "GET / HTTP/1.1\r\nHost\r\n\r\n", ErrBadRequest},
		{"space before colon", "GET / HTTP/1.1\r\nHost : h\r\n\r\n", ErrBadRequest},
		{"obs-fold", "GET / HTTP/1.1\r\nX: a\r\n b\r\n\r\n", ErrBadRequest},
		{"control byte in value", "GET / HTTP/1.1\r\nX: a\x00b\r\n\r\n", ErrBadRequest},
		{"long uri", "GET /" + long + " HTTP/1.1\r\n\r\n", ErrURITooLong},
		{"long header", "GET / HTTP/1.1\r\nX: " + long + "\r\n\r\n", ErrHeaderTooLarge},
		{"too many headers", "GET / HTTP/1.1\r\n" + manyHeaders + "\r\n", ErrHeaderTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p Parser
			done, err := p.Feed([]byte(tt.raw))
			if done || err != tt.want {
				t.Fatalf("Feed = %v, %v; want false, %v", done, err, tt.want)
			}
			// The error sticks: the connection is to be answered and closed.
			if _, again := p.Feed([]byte(tt.raw)); again != tt.want {
				t.Errorf("second Feed error = %v, want %v", again, tt.want)
			}
		})
	}
}

func TestParserIncomplete(t *testing.T) {
	var p Parser
	if done, err := p.Feed([]byte("GET / HTTP/1.1\r\nHost: h\r\n")); done || err != nil {
		t.Fatalf("Feed = %v, %v; want false, nil", done, err)
	}
	if p.HeadLen() != 0 {
		t.Errorf("HeadLen of an incomplete head = %d", p.HeadLen())
	}
}

func TestParserReset(t *testing.T) {
	var p Parser
	p.Feed([]byte("BAD\x00"))
	p.Reset()
	if done, err := p.Feed([]byte("GET /ok HTTP/1.1\r\n\r\n")); !done || err != nil {
		t.Fatalf("after Reset: Feed = %v, %v", done, err)
	}
	check(t, "path", p.Path(), "/ok")
}

func TestParserZeroAllocs(t *testing.T) {
	var p Parser
	allocs := testing.AllocsPerRun(100, func() {
		p.Reset()
		p.Feed(chromeRequest)
		_ = p.Header("Cookie")
	})
	if allocs != 0 {
		t.Errorf("allocs per request = %v, want 0", allocs)
	}
}

func FuzzParser(f *testing.F) {
	f.Add(chromeRequest, 10)
	f.Add([]byte("GET / HTTP/1.1\n\n"), 3)
	f.Add([]byte("OPTIONS * HTTP/1.0\r\nX:\r\n\r\n"), 0)
	f.Add([]byte("\r\nPOST /a?b HTTP/1.1\r\nA: b \r\n\r\nbody"), 20)
	f.Fuzz(func(t *testing.T, data []byte, cut int) {
		var whole, split Parser
		doneWhole, errWhole := whole.Feed(data)
		if errWhole != nil {
			switch errWhole {
			case ErrBadRequest, ErrURITooLong, ErrHeaderTooLarge, ErrVersion:
			default:
				t.Fatalf("unexpected error %v", errWhole)
			}
		}
		if doneWhole && (whole.HeadLen() < 1 || whole.HeadLen() > len(data)) {
			t.Fatalf("HeadLen %d out of range for %d bytes", whole.HeadLen(), len(data))
		}

		if cut < 0 {
			cut = -cut
		}
		if len(data) > 0 {
			cut %= len(data) + 1
		} else {
			cut = 0
		}
		doneSplit, errSplit := split.Feed(data[:cut])
		if errSplit == nil && !doneSplit {
			doneSplit, errSplit = split.Feed(data)
		}
		if errSplit != errWhole || doneSplit != doneWhole {
			t.Fatalf("split at %d: %v, %v; whole: %v, %v", cut, doneSplit, errSplit, doneWhole, errWhole)
		}
		if doneWhole {
			assertSameParse(t, &whole, &split)
		}
	})
}

func BenchmarkParser(b *testing.B) {
	var p Parser
	b.SetBytes(int64(len(chromeRequest)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p.Reset()
		if _, err := p.Feed(chromeRequest); err != nil {
			b.Fatal(err)
		}
	}
}

func assertSameParse(t *testing.T, a, b *Parser) {
	t.Helper()
	check(t, "method", b.Method(), string(a.Method()))
	check(t, "target", b.Target(), string(a.Target()))
	check(t, "query", b.Query(), string(a.Query()))
	if a.HeadLen() != b.HeadLen() || a.NumHeaders() != b.NumHeaders() {
		t.Fatalf("HeadLen/NumHeaders differ: %d/%d vs %d/%d", a.HeadLen(), a.NumHeaders(), b.HeadLen(), b.NumHeaders())
	}
	for i := 0; i < a.NumHeaders(); i++ {
		check(t, "header name", b.HeaderName(i), string(a.HeaderName(i)))
		check(t, "header value", b.HeaderValue(i), string(a.HeaderValue(i)))
	}
}

func check(t *testing.T, what string, got []byte, want string) {
	t.Helper()
	if string(got) != want {
		t.Errorf("%s = %q, want %q", what, got, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
