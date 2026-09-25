package http

import (
	"bytes"
	"path"
	"testing"
)

func TestCleanPath(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"/", "/"},
		{"/index.html", "/index.html"},
		{"/a/b/", "/a/b/"},
		{"/a/./b/../c", "/a/c"},
		{"//a///b", "/a/b"},
		{"/a/b/..", "/a"},
		{"/a/..", "/"},
		{"/a/../", "/"},
		{"/..", "/"},
		{"/../../etc/passwd", "/etc/passwd"},
		{"/%2e%2e/%2E%2E/etc/passwd", "/etc/passwd"},
		{"/a/..%2f..%2f..%2fetc/passwd", "/etc/passwd"},
		{"/img/%D0%BA%D0%BE%D1%82.png", "/img/кот.png"},
		{"/a%20b", "/a b"},
		{"/a+b", "/a+b"}, // "+" is a space only in form-encoded queries, not in paths
		{"/.hidden", "/.hidden"},
		{"/..a/b..", "/..a/b.."},
	}
	for _, tt := range tests {
		got, err := CleanPath(nil, []byte(tt.in))
		if err != nil {
			t.Errorf("CleanPath(%q) error: %v", tt.in, err)
			continue
		}
		if string(got) != tt.want {
			t.Errorf("CleanPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCleanPathErrors(t *testing.T) {
	for _, in := range []string{"", "*", "a/b", "/%", "/%2", "/%zz", "/%g0", "/a%00b"} {
		if _, err := CleanPath(nil, []byte(in)); err != ErrBadRequest {
			t.Errorf("CleanPath(%q) error = %v, want ErrBadRequest", in, err)
		}
	}
}

func TestCleanPathAppends(t *testing.T) {
	dst := []byte("prefix")
	got, err := CleanPath(dst, []byte("/a/../b"))
	if err != nil || string(got) != "prefix/b" {
		t.Fatalf("CleanPath = %q, %v", got, err)
	}
	got, err = CleanPath(dst, []byte("/%zz"))
	if err == nil || string(got) != "prefix" {
		t.Fatalf("on error dst must stay as it was, got %q, %v", got, err)
	}
}

func TestCleanPathZeroAllocs(t *testing.T) {
	buf := make([]byte, 0, 256)
	raw := []byte("/%2e%2e/static//css/../js/app.js")
	allocs := testing.AllocsPerRun(100, func() {
		buf, _ = CleanPath(buf[:0], raw)
	})
	if allocs != 0 {
		t.Errorf("allocs = %v, want 0", allocs)
	}
}

func FuzzCleanPath(f *testing.F) {
	for _, s := range []string{"/", "/../..", "/%2e%2e/x", "//a/./b/../", "/a%2f..%2f..", "/..%00"} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		got, err := CleanPath(nil, raw)
		if err != nil {
			return
		}
		if len(got) == 0 || got[0] != '/' {
			t.Fatalf("CleanPath(%q) = %q: not rooted", raw, got)
		}
		trimmed := bytes.TrimSuffix(got, []byte("/"))
		if len(trimmed) > 0 {
			for _, segment := range bytes.Split(trimmed[1:], []byte("/")) {
				if len(segment) == 0 || string(segment) == "." || string(segment) == ".." {
					t.Fatalf("CleanPath(%q) = %q: bad segment %q", raw, got, segment)
				}
			}
		}
		if want := path.Clean(string(got)); string(trimmed) != want && !(want == "/" && len(trimmed) == 0) {
			t.Fatalf("CleanPath(%q) = %q, path.Clean gives %q", raw, got, want)
		}
	})
}

func BenchmarkCleanPath(b *testing.B) {
	buf := make([]byte, 0, 256)
	raw := []byte("/%2e%2e/static//css/../js/app.js")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, _ = CleanPath(buf[:0], raw)
	}
}
