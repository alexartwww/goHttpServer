package http

type Parser struct {
	buf   []byte
	state state
	pos   int
	err   *Error

	lit int // position inside the "HTTP/" literal

	method span
	target span
	path   span
	query  span
	major  uint8
	minor  uint8

	headers  [MaxHeaders]field
	nHeaders int
	valueEnd int // end of the current header value without trailing whitespace
}

const (
	MaxHeadSize = 8 << 10
	MaxHeaders  = 64
)

type state uint8

const (
	stMethod         state = iota // GET
	stTargetStart                 // first byte of the target: "/" or "*"
	stPath                        // /index.html
	stQuery                       // ?a=1&b=2
	stVersion                     // HTTP/
	stVersionMajor                // 1
	stVersionDot                  // .
	stVersionMinor                // 1
	stRequestLineEnd              // CR LF or a bare LF
	stRequestLineLF               // LF after CR
	stHeaderStart                 // a header name, or the empty line that ends the head
	stHeaderName                  // Host
	stHeaderValueOWS              // whitespace after the colon
	stHeaderValue                 // example.com
	stHeaderLF                    // LF after CR
	stHeadEndLF                   // LF of the empty line
	stDone
)

type span struct {
	start, end int
}

type field struct {
	name, value span
}

type Error struct {
	Status int
	Reason string
}

func (e *Error) Error() string {
	return e.Reason
}

var (
	ErrBadRequest     = &Error{400, "Bad Request"}
	ErrURITooLong     = &Error{414, "URI Too Long"}
	ErrHeaderTooLarge = &Error{431, "Request Header Fields Too Large"}
	ErrVersion        = &Error{505, "HTTP Version Not Supported"}
)

const versionLiteral = "HTTP/"

func (p *Parser) Reset() {
	*p = Parser{}
}

func (p *Parser) Feed(buf []byte) (bool, error) {
	p.buf = buf
	if p.err != nil {
		return false, p.err
	}
	if p.state == stDone {
		return true, nil
	}

	for i := p.pos; i < len(buf); i++ {
		if i >= MaxHeadSize {
			if p.state == stPath || p.state == stQuery {
				return false, p.fail(ErrURITooLong)
			}
			return false, p.fail(ErrHeaderTooLarge)
		}
		c := buf[i]

		switch p.state {
		case stMethod:
			switch {
			case isToken(c):
			case c == ' ' && i > p.method.start:
				p.method.end = i
				p.target.start = i + 1
				p.state = stTargetStart
			case (c == '\r' || c == '\n') && i == p.method.start:
				// Empty lines before the request line are ignored (RFC 9112, 2.2).
				p.method.start = i + 1
			default:
				return false, p.fail(ErrBadRequest)
			}

		case stTargetStart:
			// Only origin-form ("/path?query") and asterisk-form ("*") are served.
			if c != '/' && c != '*' {
				return false, p.fail(ErrBadRequest)
			}
			p.path.start = i
			p.state = stPath

		case stPath:
			switch {
			case c == '?':
				p.path.end = i
				p.query.start = i + 1
				p.state = stQuery
			case c == ' ':
				p.path.end = i
				p.query = span{i, i}
				p.target.end = i
				p.state = stVersion
			case !isVisible(c):
				return false, p.fail(ErrBadRequest)
			}

		case stQuery:
			switch {
			case c == ' ':
				p.query.end = i
				p.target.end = i
				p.state = stVersion
			case !isVisible(c):
				return false, p.fail(ErrBadRequest)
			}

		case stVersion:
			if c != versionLiteral[p.lit] {
				return false, p.fail(ErrBadRequest)
			}
			p.lit++
			if p.lit == len(versionLiteral) {
				p.state = stVersionMajor
			}

		case stVersionMajor:
			if !isDigit(c) {
				return false, p.fail(ErrBadRequest)
			}
			p.major = c - '0'
			p.state = stVersionDot

		case stVersionDot:
			if c != '.' {
				return false, p.fail(ErrBadRequest)
			}
			p.state = stVersionMinor

		case stVersionMinor:
			if !isDigit(c) {
				return false, p.fail(ErrBadRequest)
			}
			p.minor = c - '0'
			if p.major != 1 || p.minor > 1 {
				return false, p.fail(ErrVersion)
			}
			p.state = stRequestLineEnd

		case stRequestLineEnd:
			switch c {
			case '\r':
				p.state = stRequestLineLF
			case '\n':
				p.state = stHeaderStart
			default:
				return false, p.fail(ErrBadRequest)
			}

		case stRequestLineLF:
			if c != '\n' {
				return false, p.fail(ErrBadRequest)
			}
			p.state = stHeaderStart

		case stHeaderStart:
			switch {
			case c == '\r':
				p.state = stHeadEndLF
			case c == '\n':
				return p.done(i)
			case isToken(c):
				if p.nHeaders == MaxHeaders {
					return false, p.fail(ErrHeaderTooLarge)
				}
				p.headers[p.nHeaders].name.start = i
				p.state = stHeaderName
			default:
				// Including SP and HTAB: line folding is obsolete and must be rejected (RFC 9112, 5.2).
				return false, p.fail(ErrBadRequest)
			}

		case stHeaderName:
			switch {
			case isToken(c):
			case c == ':':
				p.headers[p.nHeaders].name.end = i
				p.state = stHeaderValueOWS
			default:
				// Including whitespace before the colon (RFC 9112, 5.1).
				return false, p.fail(ErrBadRequest)
			}

		case stHeaderValueOWS:
			switch {
			case c == ' ' || c == '\t':
			case c == '\r':
				p.headers[p.nHeaders].value = span{i, i}
				p.state = stHeaderLF
			case c == '\n':
				p.headers[p.nHeaders].value = span{i, i}
				p.nHeaders++
				p.state = stHeaderStart
			case isFieldByte(c):
				p.headers[p.nHeaders].value.start = i
				p.valueEnd = i + 1
				p.state = stHeaderValue
			default:
				return false, p.fail(ErrBadRequest)
			}

		case stHeaderValue:
			switch {
			case c == ' ' || c == '\t':
				// Kept only if more value bytes follow: trailing whitespace is not part of the value.
			case c == '\r':
				p.headers[p.nHeaders].value.end = p.valueEnd
				p.state = stHeaderLF
			case c == '\n':
				p.headers[p.nHeaders].value.end = p.valueEnd
				p.nHeaders++
				p.state = stHeaderStart
			case isFieldByte(c):
				p.valueEnd = i + 1
			default:
				return false, p.fail(ErrBadRequest)
			}

		case stHeaderLF:
			if c != '\n' {
				return false, p.fail(ErrBadRequest)
			}
			p.nHeaders++
			p.state = stHeaderStart

		case stHeadEndLF:
			if c != '\n' {
				return false, p.fail(ErrBadRequest)
			}
			return p.done(i)
		}
	}

	p.pos = len(buf)
	return false, nil
}

func (p *Parser) done(i int) (bool, error) {
	p.pos = i + 1
	p.state = stDone
	return true, nil
}

func (p *Parser) fail(err *Error) error {
	p.err = err
	return err
}

func (p *Parser) HeadLen() int {
	if p.state != stDone {
		return 0
	}
	return p.pos
}

func (p *Parser) Method() []byte { return p.bytes(p.method) }

func (p *Parser) Target() []byte { return p.bytes(p.target) }

func (p *Parser) Path() []byte { return p.bytes(p.path) }

func (p *Parser) Query() []byte { return p.bytes(p.query) }

func (p *Parser) Version() (major, minor int) { return int(p.major), int(p.minor) }

func (p *Parser) NumHeaders() int { return p.nHeaders }

func (p *Parser) HeaderName(i int) []byte { return p.bytes(p.headers[i].name) }

func (p *Parser) HeaderValue(i int) []byte { return p.bytes(p.headers[i].value) }

func (p *Parser) Header(name string) []byte {
	for i := 0; i < p.nHeaders; i++ {
		if equalFold(p.bytes(p.headers[i].name), name) {
			return p.bytes(p.headers[i].value)
		}
	}
	return nil
}

func (p *Parser) VisitQuery(fn func(name, value []byte) bool) {
	visitPairs(p.Query(), '&', fn)
}

func (p *Parser) VisitCookies(fn func(name, value []byte) bool) {
	for i := 0; i < p.nHeaders; i++ {
		if equalFold(p.bytes(p.headers[i].name), "Cookie") {
			if !visitPairs(p.bytes(p.headers[i].value), ';', fn) {
				return
			}
		}
	}
}

func (p *Parser) bytes(s span) []byte {
	return p.buf[s.start:s.end]
}

func visitPairs(b []byte, sep byte, fn func(name, value []byte) bool) bool {
	for len(b) > 0 {
		pair := b
		if i := indexByte(b, sep); i >= 0 {
			pair, b = b[:i], b[i+1:]
		} else {
			b = nil
		}
		pair = trimSpace(pair)
		if len(pair) == 0 {
			continue
		}
		name, value := pair, pair[len(pair):]
		if i := indexByte(pair, '='); i >= 0 {
			name, value = pair[:i], pair[i+1:]
		}
		if !fn(name, value) {
			return false
		}
	}
	return true
}

func indexByte(b []byte, c byte) int {
	for i, v := range b {
		if v == c {
			return i
		}
	}
	return -1
}

func trimSpace(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t') {
		b = b[1:]
	}
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == '\t') {
		b = b[:len(b)-1]
	}
	return b
}

func equalFold(b []byte, s string) bool {
	if len(b) != len(s) {
		return false
	}
	for i := 0; i < len(b); i++ {
		if lower(b[i]) != lower(s[i]) {
			return false
		}
	}
	return true
}

func lower(c byte) byte {
	if 'A' <= c && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

var (
	tokenTable [256]bool
	fieldTable [256]bool
)

func init() {
	for c := 'a'; c <= 'z'; c++ {
		tokenTable[c] = true
	}
	for c := 'A'; c <= 'Z'; c++ {
		tokenTable[c] = true
	}
	for c := '0'; c <= '9'; c++ {
		tokenTable[c] = true
	}
	for _, c := range "!#$%&'*+-.^_`|~" {
		tokenTable[c] = true
	}
	for c := 0x21; c <= 0xFF; c++ {
		fieldTable[c] = c != 0x7F
	}
}

func isToken(c byte) bool     { return tokenTable[c] }
func isFieldByte(c byte) bool { return fieldTable[c] }
func isVisible(c byte) bool   { return c > 0x20 && c < 0x7F }
func isDigit(c byte) bool     { return '0' <= c && c <= '9' }
