package http

import (
	"strconv"
)

// GET /radio/listen/ HTTP/1.1
// Host: artem-aleksashkin
// Connection: keep-alive
// Cache-Control: max-age=0
// DNT: 1
// Upgrade-Insecure-Requests: 1
// User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.182 Safari/537.36
// Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9
// Referer: http://artem-aleksashkin/
// Accept-Encoding: gzip, deflate
// Accept-Language: ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7
// Cookie: id=ca2a568d-cdf0-4658-b98c-320a9f1b0eb6; geography=1; timezone=Europe%2FMoscow; language=ru; language-data=ru%2Cen; currency=rub; user=c4ca4238a0b923820dcc509a6f75849b

// HTTP/1.1 200 OK
// Server: nginx/1.14.0 (Ubuntu)
// Date: Sun, 18 Jul 2021 01:13:24 GMT
// Content-Type: text/html;charset=UTF-8
// Transfer-Encoding: chunked
// Connection: keep-alive
// Set-Cookie: geography=1; expires=Sun, 01-Aug-2021 01:13:24 GMT; Max-Age=1209600; path=/; domain=artem-aleksashkin
// Content-Encoding: gzip

type Response struct {
	Protocol string
	Version string
	Code uint16
	Status string
	Headers []Header
	Cookies []Cookie
	Body []byte
}

var delimiter = []byte("\r\n")

func (response *Response) Build() []byte {
	result := response.AppendHead(make([]byte, 0, 256+len(response.Body)))
	return append(result, response.Body...)
}

// AppendHead appends the status line, headers and the empty line, without the body.
func (response *Response) AppendHead(result []byte) []byte {

	// HTTP/1.1 200 OK\r\n
	result = append(result, response.Protocol...)
	result = append(result, "/"...)
	result = append(result, response.Version...)
	result = append(result, " "...)
	result = strconv.AppendInt(result, int64(response.Code), 10)
	result = append(result, " "...)
	result = append(result, response.Status...)
	result = append(result, delimiter...)

	// Headers
	for _, header := range response.Headers {
		// Server: nginx/1.14.0 (Ubuntu)\r\n
		result = append(result, header.Name...)
		result = append(result, ": "...)
		result = append(result, header.Value...)
		result = append(result, delimiter...)
	}

	// Cookies
	for _, cookie := range response.Cookies {
		// Set-Cookie: geography=1; expires=Sun, 01-Aug-2021 01:13:24 GMT; Max-Age=1209600; path=/; domain=artem-aleksashkin\r\n
		result = append(result, "Set-Cookie: "...)
		result = append(result, cookie.Name...)
		result = append(result, "="...)
		result = append(result, cookie.Value...)
		if !cookie.Expires.IsZero() {
			result = append(result, "; expires="...)
			result = append(result, []byte(FormatTime(cookie.Expires))...)
		}
		if cookie.MaxAge != 0 {
			result = append(result, "; Max-Age="...)
			result = strconv.AppendUint(result, cookie.MaxAge, 10)
		}
		if cookie.Path != "" {
			result = append(result, "; path="...)
			result = append(result, cookie.Path...)
		}
		if cookie.Domain != "" {
			result = append(result, "; domain="...)
			result = append(result, cookie.Domain...)
		}
		if cookie.Secure {
			result = append(result, "; secure"...)
		}
		if cookie.HttpOnly {
			result = append(result, "; httponly"...)
		}
		if cookie.SameSite != "" {
			result = append(result, "; samesite="...)
			result = append(result, cookie.SameSite...)
		}
		result = append(result, delimiter...)
	}

	// The empty line ends the head even when there is no body
	return append(result, delimiter...)
}
