package main

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alexartwww/goHttpServer/internal/file"
	"github.com/alexartwww/goHttpServer/internal/http"
)

// Small files are kept in memory with ready headers. An entry is trusted for
// cacheValid, after that the next request goes to the disk and refreshes it.
const cacheValid = time.Second
const cacheMaxBytes = 64 << 20

type cacheEntry struct {
	etag     string
	modified time.Time
	headers  []http.Header // Last-Modified and ETag go first: only they are sent with 304
	body     []byte
	loaded   time.Time
}

// One cache per virtual host, the size limit is shared by all of them.
type fileCache struct {
	entries sync.Map // clean request path -> *cacheEntry
}

var fileCacheBytes atomic.Int64

func (c *fileCache) get(path []byte, now time.Time) *cacheEntry {
	value, ok := c.entries.Load(string(path))
	if !ok {
		return nil
	}
	entry := value.(*cacheEntry)
	if now.Sub(entry.loaded) > cacheValid {
		return nil
	}
	return entry
}

func (c *fileCache) put(path string, info *file.File, body []byte, now time.Time) {
	entry := &cacheEntry{
		etag:     info.ETag,
		modified: info.DateTime,
		headers:  fileHeaders(info),
		body:     body,
		loaded:   now,
	}
	previous, loaded := c.entries.Swap(path, entry)
	if loaded {
		fileCacheBytes.Add(-int64(len(previous.(*cacheEntry).body)))
	}
	if fileCacheBytes.Add(int64(len(body))) > cacheMaxBytes {
		c.entries.CompareAndDelete(path, entry)
		fileCacheBytes.Add(-int64(len(body)))
	}
}

func fileHeaders(info *file.File) []http.Header {
	headers := make([]http.Header, 0, 4)
	if !info.DateTime.IsZero() {
		headers = append(headers, http.Header{Name: "Last-Modified", Value: http.FormatTime(info.DateTime)})
	}
	if info.ETag != "" {
		headers = append(headers, http.Header{Name: "ETag", Value: info.ETag})
	}
	if info.Mimetype != "" {
		headers = append(headers, http.Header{Name: "Content-Type", Value: info.Mimetype})
	}
	return append(headers, http.Header{Name: "Content-Length", Value: strconv.FormatInt(info.Size, 10)})
}

func validators(headers []http.Header) []http.Header {
	n := 0
	for n < len(headers) && (headers[n].Name == "Last-Modified" || headers[n].Name == "ETag") {
		n++
	}
	return headers[:n]
}
