package http

func CleanPath(dst, path []byte) ([]byte, error) {
	if len(path) == 0 || path[0] != '/' {
		return dst, ErrBadRequest
	}

	start := len(dst)
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '%' {
			if i+2 >= len(path) {
				return dst[:start], ErrBadRequest
			}
			hi, okHi := unhex(path[i+1])
			lo, okLo := unhex(path[i+2])
			if !okHi || !okLo {
				return dst[:start], ErrBadRequest
			}
			c = hi<<4 | lo
			i += 2
		}
		if c == 0 {
			return dst[:start], ErrBadRequest
		}
		dst = append(dst, c)
	}

	clean := cleanInPlace(dst[start:])
	return dst[:start+len(clean)], nil
}

func cleanInPlace(buf []byte) []byte {
	n := len(buf)
	trailingSlash := n > 1 && buf[n-1] == '/'

	w := 1 // buf[0] is the root "/"
	for r := 1; r < n; {
		switch {
		case buf[r] == '/':
			r++
		case buf[r] == '.' && (r+1 == n || buf[r+1] == '/'):
			r++
		case buf[r] == '.' && buf[r+1] == '.' && (r+2 == n || buf[r+2] == '/'):
			r += 2
			// Back to the previous slash, but never above the root.
			for w > 1 {
				w--
				if buf[w] == '/' {
					break
				}
			}
		default:
			if w > 1 {
				buf[w] = '/'
				w++
			}
			for ; r < n && buf[r] != '/'; r++ {
				buf[w] = buf[r]
				w++
			}
		}
	}

	if trailingSlash && w > 1 {
		buf[w] = '/'
		w++
	}
	return buf[:w]
}

func unhex(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}
