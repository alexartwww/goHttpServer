package http

import (
	"strconv"
	"time"
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
	result := make([]byte, 0)

	// HTTP/1.1 200 OK\r\n
	result = append(result, []byte(response.Protocol)...)
	result = append(result, []byte("/")...)
	result = append(result, []byte(response.Version)...)
	result = append(result, []byte(" ")...)
	result = append(result, []byte(strconv.Itoa(int(response.Code)))...)
	result = append(result, []byte(" ")...)
	result = append(result, []byte(response.Status)...)
	result = append(result, delimiter...)

	// Headers
	for _, header := range response.Headers {
		// Server: nginx/1.14.0 (Ubuntu)\r\n
		result = append(result, []byte(header.Name)...)
		result = append(result, []byte(": ")...)
		result = append(result, []byte(header.Value)...)
		result = append(result, delimiter...)
	}

	// Cookies
	for _, cookie := range response.Cookies {
		// Set-Cookie: geography=1; expires=Sun, 01-Aug-2021 01:13:24 GMT; Max-Age=1209600; path=/; domain=artem-aleksashkin\r\n
		result = append(result, []byte("Set-Cookie: ")...)
		result = append(result, []byte(cookie.Name)...)
		result = append(result, []byte("=")...)
		result = append(result, []byte(cookie.Value)...)
		if !cookie.Expires.IsZero() {
			result = append(result, []byte("; expires=")...)
			geoLocation, _ := time.LoadLocation("GMT")
			result = append(result, []byte(cookie.Expires.In(geoLocation).Format(time.RFC850))...)
		}
		if cookie.MaxAge != 0 {
			result = append(result, []byte("; Max-Age=")...)
			result = append(result, []byte(strconv.Itoa(int(cookie.MaxAge)))...)
		}
		if cookie.Path != "" {
			result = append(result, []byte("; path=")...)
			result = append(result, []byte(cookie.Path)...)
		}
		if cookie.Domain != "" {
			result = append(result, []byte("; domain=")...)
			result = append(result, []byte(cookie.Domain)...)
		}
		if cookie.Secure {
			result = append(result, []byte("; secure")...)
		}
		if cookie.HttpOnly {
			result = append(result, []byte("; httponly")...)
		}
		if cookie.SameSite != "" {
			result = append(result, []byte("; samesite=")...)
			result = append(result, []byte(cookie.SameSite)...)
		}
		result = append(result, delimiter...)
	}

	// Body
	if len(response.Body) > 0 {
		result = append(result, delimiter...)
		result = append(result, response.Body...)
	}
	return result
}
