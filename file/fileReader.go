package file

import (
	"io"
	"os"
	"path/filepath"
	"time"
)

type File struct {
	Name string
	Directory bool
	Exist bool
	Readable bool
	Size int64
	DateTime time.Time
	Mimetype string
	Encoding string
	ETag string

	info os.FileInfo
	infoErr error
}

// image formats and magic numbers
var mimeTypes = map[string]string{
	".aac": "audio/aac",
	".abw": "application/x-abiword",
	".arc": "application/x-freearc",
	".avi": "video/x-msvideo",
	".azw": "application/vnd.amazon.ebook",
	".bin": "application/octet-stream",
	".bmp": "image/bmp",
	".bz": "application/x-bzip",
	".bz2": "application/x-bzip2",
	".csh": "application/x-csh",
	".css": "text/css",
	".csv": "text/csv",
	".doc": "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".eot": "application/vnd.ms-fontobject",
	".epub": "application/epub+zip",
	".gz": "application/gzip",
	".gif": "image/gif",
	".htm": "text/html",
	".html": "text/html",
	".ico": "image/vnd.microsoft.icon",
	".ics": "text/calendar",
	".jar": "application/java-archive",
	".jpeg": "image/jpeg",
	".jpg": "image/jpeg",
	".js": "text/javascript",
	".json": "application/json",
	".jsonld": "application/ld+json",
	".mid": "audio/midi",
	".midi": "audio/midi",
	".mjs": "text/javascript",
	".mp3": "audio/mpeg",
	".mpeg": "video/mpeg",
	".mpkg": "application/vnd.apple.installer+xml",
	".odp": "application/vnd.oasis.opendocument.presentation",
	".ods": "application/vnd.oasis.opendocument.spreadsheet",
	".odt": "application/vnd.oasis.opendocument.text",
	".oga": "audio/ogg",
	".ogv": "video/ogg",
	".ogx": "application/ogg",
	".opus": "audio/opus",
	".otf": "font/otf",
	".png": "image/png",
	".pdf": "application/pdf",
	".php": "application/php",
	".ppt": "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".rar": "application/vnd.rar",
	".rtf": "application/rtf",
	".sh": "application/x-sh",
	".svg": "image/svg+xml",
	".swf": "application/x-shockwave-flash",
	".tar": "application/x-tar",
	".tif": "image/tiff",
	".tiff": "image/tiff",
	".ts": "video/mp2t",
	".ttf": "font/ttf",
	".txt": "text/plain",
	".vsd": "application/vnd.visio",
	".wav": "audio/wav",
	".weba": "audio/webm",
	".webm": "video/webm",
	".webp": "image/webp",
	".woff": "font/woff",
	".woff2": "font/woff2",
	".xhtml": "application/xhtml+xml",
	".xls": "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".xml": "application/xml",
	".xul": "application/vnd.mozilla.xul+xml",
	".zip": "application/zip",
	".3gp": "video/3gpp",
	".3g2": "video/3gpp2",
	".7z": "application/x-7z-compressed",
}

func (file *File) getInfo() {
	file.info, file.infoErr = os.Stat(file.Name)
}

func (file *File) checkExist() bool {
	if file.infoErr != nil {
		if os.IsNotExist(file.infoErr) {
			return false
		}
	}
	return true
}

func (file *File) checkReadable() bool {
	f, err := os.Open(file.Name)
	if err != nil {
		return false
	}
	if file.info.Size() > 0 {
		b1 := make([]byte, 1)
		_, err2 := f.Read(b1)
		if err2 != nil {
			return false
		}
	}
	f.Close()
	return true
}

func (file *File) checkDirectory() bool {
	return file.info.IsDir()
}

func (file *File) getSize() int64 {
	return file.info.Size()
}

func (file *File) getDateTime() time.Time {
	return file.info.ModTime()
}

func (file *File) getMimetype() string {
	ext := filepath.Ext(file.Name)
	_, ok := mimeTypes[ext]
	if ok {
		return mimeTypes[ext]
	}
	return ""
}

func (file *File) getEncoding() string {
	return ""
}

func (file *File) getETag() string {
	return ""
}

func (file *File) Info(name string) {
	file.Name = name
	file.getInfo()

	file.Exist = file.checkExist()
	if file.Exist {
		file.Directory = file.checkDirectory()
		if !file.Directory {
			file.Readable = file.checkReadable()
			if file.Readable {
				file.Size = file.getSize()
				file.DateTime = file.getDateTime()
				file.Mimetype = file.getMimetype()
				file.Encoding = file.getEncoding()
				file.ETag = file.getETag()
			}
		}
	}
}

func (file *File) Read() []byte {
	result := make([]byte, 0)

	f, err := os.Open(file.Name)
	if err != nil {
		return result
	}
	buff := make([]byte, 32*1024)
	for {
		n, err2 := f.Read(buff)
		if n > 0 {
			result = append(result, buff[:n]...)
		}
		if err2 == io.EOF {
			break
		} else if err != nil {
			break
		}
	}
	f.Close()
	return result
}
