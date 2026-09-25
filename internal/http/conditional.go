package http

import "time"

// NotModified reports whether a 304 can be sent instead of the file (RFC 9110, 13.2.2):
// If-None-Match wins, If-Modified-Since is checked only without it.
func NotModified(request *Parser, etag string, modified time.Time) bool {
	if match := request.Header("If-None-Match"); match != nil {
		return etag != "" && etagMatch(match, etag)
	}
	if since := request.Header("If-Modified-Since"); since != nil && !modified.IsZero() {
		t, ok := ParseTime(since)
		return ok && !modified.Truncate(time.Second).After(t)
	}
	return false
}

// etagMatch is the weak comparison: W/"x" and "x" are the same entity tag.
func etagMatch(list []byte, etag string) bool {
	etag = trimWeak(etag)
	for len(list) > 0 {
		item := list
		if i := indexByte(list, ','); i >= 0 {
			item, list = list[:i], list[i+1:]
		} else {
			list = nil
		}
		item = trimSpace(item)
		if string(item) == "*" || trimWeak(string(item)) == etag {
			return true
		}
	}
	return false
}

func trimWeak(etag string) string {
	if len(etag) > 2 && etag[:2] == "W/" {
		return etag[2:]
	}
	return etag
}
