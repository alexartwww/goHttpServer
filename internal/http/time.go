package http

import "time"

const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

// RFC 9110, 5.6.7: recipients must also accept the obsolete RFC 850 and asctime formats.
var timeFormats = []string{TimeFormat, "Monday, 02-Jan-06 15:04:05 GMT", time.ANSIC}

func FormatTime(t time.Time) string {
	return t.UTC().Format(TimeFormat)
}

func ParseTime(b []byte) (time.Time, bool) {
	s := string(b)
	for _, format := range timeFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
