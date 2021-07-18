package file

type File struct {
	Name string
	Exist bool
	Readable bool
	Size uint64
	Date string
	Mimetype string
	Etag string
}

