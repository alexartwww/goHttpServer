package http

type Cookie struct {
	Name string
	Value string
	Expires string
	MaxAge uint64
	Path string
	Domain string
	Secure bool
	HttpOnly bool
	SameSite string
}
