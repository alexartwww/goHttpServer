package http

// RFC 9112, 9.3: HTTP/1.1 is persistent unless "close", HTTP/1.0 only with "keep-alive".
func (p *Parser) KeepAlive() bool {
	connection := p.Header("Connection")
	if p.major == 1 && p.minor == 0 {
		return hasToken(connection, "keep-alive")
	}
	return !hasToken(connection, "close")
}

func (p *Parser) HasBody() bool {
	if p.Header("Transfer-Encoding") != nil {
		return true
	}
	length := p.Header("Content-Length")
	return length != nil && string(trimSpace(length)) != "0"
}

func hasToken(list []byte, token string) bool {
	for len(list) > 0 {
		item := list
		if i := indexByte(list, ','); i >= 0 {
			item, list = list[:i], list[i+1:]
		} else {
			list = nil
		}
		if equalFold(trimSpace(item), token) {
			return true
		}
	}
	return false
}
