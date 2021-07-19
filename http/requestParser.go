package http

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

type Request struct {
	Method string
	Iri string
	Path string
	Params []Param
	Protocol string
	Version string
	Headers []Header
	Cookies []Cookie
	Body []byte

	flagMethod bool
	flagIri bool
	flagPath bool
	flagParamName bool
	flagParamValue bool
	flagProtocol bool
	flagVersion bool
	flagHeaderName bool
	flagHeaderValue bool
	flagHeaderValueSpace bool
	flagCookieName bool
	flagCookieValue bool
	flagBody bool
}

func (request *Request) switchFlags(v byte) bool {
	if !request.flagMethod && !request.flagIri && !request.flagProtocol && !request.flagVersion && !request.flagHeaderName && !request.flagHeaderValue && !request.flagBody {
		request.flagMethod = true
		return false
	}
	if !request.flagBody && v == byte('\r') {
		return true
	}
	if request.flagMethod && v == byte(' ') {
		request.flagMethod = false
		request.flagIri = true
		request.flagPath = true
		return true
	}

	if request.flagPath && v == byte('?') {
		request.flagPath = false
		request.flagParamName = true
		request.flagParamValue = false
		request.Params = append(request.Params, Param{})
		return false
	}
	if request.flagParamName && v == byte('=') {
		request.flagParamName = false
		request.flagParamValue = true
		return false
	}
	if request.flagParamValue && v == byte('&') {
		request.flagParamName = true
		request.flagParamValue = false
		request.Params = append(request.Params, Param{})
		return false
	}
	if request.flagIri && v == byte(' ') {
		request.flagIri = false
		request.flagPath = false
		request.flagParamName = false
		request.flagParamValue = false
		request.flagProtocol = true
		return true
	}
	if request.flagProtocol && v == byte('/') {
		request.flagProtocol = false
		request.flagVersion = true
		return true
	}
	if request.flagVersion && v == byte('\n') {
		request.flagVersion = false
		request.flagHeaderName = true
		request.Headers = append(request.Headers, Header{})
		return true
	}
	if request.flagHeaderName && v == byte(':') {
		request.flagHeaderName = false
		request.flagHeaderValueSpace = true
		request.flagHeaderValue = true
		if len(request.Headers) > 0 && request.Headers[len(request.Headers)-1].Name == "Cookie" {
			request.flagCookieName = true
			request.Cookies = append(request.Cookies, Cookie{})
		}
		return true
	}
	if request.flagHeaderValueSpace && v == byte(' ') {
		request.flagHeaderValueSpace = false
		return true
	}
	if request.flagHeaderValueSpace && v != byte(' ') {
		request.flagHeaderValueSpace = false
	}
	if request.flagCookieName && v == byte('=') {
		request.flagCookieName = false
		request.flagCookieValue = true
	}
	if request.flagCookieValue && v == byte(';') {
		request.flagCookieValue = false
		request.flagCookieName = true
		request.Cookies = append(request.Cookies, Cookie{})
	}
	if request.flagHeaderValue && v == byte('\n') {
		request.flagHeaderValue = false
		request.flagHeaderName = true
		request.flagCookieName = false
		request.flagCookieValue = false
		request.Headers = append(request.Headers, Header{})
		return true
	}
	if (request.flagHeaderName) && v == byte('\n') {
		request.flagHeaderName = false
		request.flagHeaderValue = false
		request.Headers = request.Headers[:len(request.Headers)-1]
		request.flagBody = true
		return true
	}
	return false
}

func (request *Request) readChar(v byte) {
	if request.flagMethod {
		request.Method = request.Method + string(v)
	}
	if request.flagIri {
		request.Iri = request.Iri + string(v)
	}
	if request.flagPath {
		request.Path = request.Path + string(v)
	}
	if request.flagParamName && v != byte('?') && v != byte('&') {
		request.Params[len(request.Params)-1].Name = request.Params[len(request.Params)-1].Name + string(v)
	}
	if request.flagParamValue && v != byte('=') {
		request.Params[len(request.Params)-1].Value = request.Params[len(request.Params)-1].Value + string(v)
	}
	if request.flagProtocol {
		request.Protocol = request.Protocol + string(v)
	}
	if request.flagVersion {
		request.Version = request.Version + string(v)
	}
	if request.flagHeaderName {
		request.Headers[len(request.Headers)-1].Name = request.Headers[len(request.Headers)-1].Name + string(v)
	}
	if request.flagHeaderValue {
		request.Headers[len(request.Headers)-1].Value = request.Headers[len(request.Headers)-1].Value + string(v)
	}
	if request.flagCookieName && v != byte(' ') && v != byte(';') && v != byte('=') {
		request.Cookies[len(request.Cookies)-1].Name = request.Cookies[len(request.Cookies)-1].Name + string(v)
	}
	if request.flagCookieValue && v != byte(' ') && v != byte(';') && v != byte('=') {
		request.Cookies[len(request.Cookies)-1].Value = request.Cookies[len(request.Cookies)-1].Value + string(v)
	}
	if request.flagBody {
		request.Body = append(request.Body, v)
	}
}

func (request *Request) Parse(buff []byte) {
	for _, v := range buff {
		if request.switchFlags(v) {
			continue
		}
		request.readChar(v)
	}
	request.flagBody = false
}
