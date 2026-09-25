package http

import "time"

type Cookie struct {
	Name string
	Value string
	Expires time.Time
	MaxAge uint64
	Path string
	Domain string
	Secure bool
	HttpOnly bool
	SameSite string
}
