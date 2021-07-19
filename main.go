package main

import (
	"alexartwww/goHttpServer/file"
	"alexartwww/goHttpServer/http"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

// Config
// =======================================================
var address = "localhost:4321"
var home = "./www"
var indexes = []string{"", "index.html", "index.htm"}
// =======================================================

func main() {
	fmt.Println("Listen http://" + address)
	listener, _ := net.Listen("tcp", address)
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	var currentTime = time.Now()
	geoLocation, _ := time.LoadLocation("GMT")

	frameLen := 32
	frame := make([]byte, frameLen)
	buff := make([]byte, 0)
	for {
		readLen, err := conn.Read(frame)
		if err != nil {
			if err != io.EOF {
				fmt.Println("read error:", err)
			}
			break
		}
		buff = append(buff, frame[:readLen]...)
		if readLen < frameLen {
			break
		}
	}

	request := http.Request{}
	request.Parse(buff)

	reader := file.File{}

	for _, index := range indexes {
		fileLocation := home + request.Path
		if fileLocation[len(fileLocation)-1:] != "/" && index != "" {
			fileLocation = fileLocation + "/"
		}
		fileLocation = fileLocation + index
		reader.Info(fileLocation)
		if reader.Exist {
			if reader.Directory {
				continue
			}
			if reader.Readable {
				response200 := http.Response{
					Protocol: request.Protocol,
					Version: request.Version,
					Code: 200,
					Status: "OK",
					Headers: []http.Header {
						{Name: "Server", Value: "goHttpServer/0.0.1"},
						{Name: "Date", Value: reader.DateTime.In(geoLocation).Format(time.RFC1123)},
					},
				}
				if reader.ETag != "" {
					response200.Headers = append(response200.Headers, http.Header{Name: "ETag", Value: reader.ETag})
				}
				if reader.Size > 0 {
					response200.Headers = append(response200.Headers, http.Header{Name: "Content-Size", Value: strconv.Itoa(int(reader.Size))})
				}
				if reader.Mimetype != "" {
					response200.Headers = append(response200.Headers, http.Header{Name: "Content-Type", Value: reader.Mimetype})
				}
				response200.Body = reader.Read()
				conn.Write(response200.Build())
			} else {
				response403 := http.Response{
					Protocol: request.Protocol,
					Version: request.Version,
					Code: 403,
					Status: "Forbidden",
					Headers: []http.Header {
						{Name: "Server", Value: "goHttpServer/0.0.1"},
					},
					Body: []byte("<html><head><title>403 Forbidden</title></head><body><h1 style=\"text-align: center;\">404 Not Found</h1><hr><p style=\"text-align: center;\">goHttpServer/0.0.1 " + currentTime.In(geoLocation).Format(time.RFC1123) + "</p></body></html>"),
				}
				conn.Write(response403.Build())
			}
			return
		}
	}

	response404 := http.Response{
		Protocol: request.Protocol,
		Version: request.Version,
		Code: 404,
		Status: "Not Found",
		Headers: []http.Header {
			{Name: "Server", Value: "goHttpServer/0.0.1"},
		},
		Body: []byte("<html><head><title>404 Not Found</title></head><body><h1 style=\"text-align: center;\">404 Not Found</h1><hr><p style=\"text-align: center;\">goHttpServer/0.0.1 " + currentTime.In(geoLocation).Format(time.RFC1123) + "</p></body></html>"),
	}
	conn.Write(response404.Build())
	return
}

// This will goes to tests
// =======================================================================

func testRequest() {
	buff := []byte("GET /?a=1&b=2&FFSDF=asdasd HTTP/1.1\nHost: artem-aleksashkin\nConnection: keep-alive\nCache-Control: max-age=0\nDNT: 1\nUpgrade-Insecure-Requests: 1\nUser-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.182 Safari/537.36\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9\nReferer: http://artem-aleksashkin/\nAccept-Encoding: gzip, deflate\nAccept-Language: ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7\nCookie: id=ca2a568d-cdf0-4658-b98c-320a9f1b0eb6; geography=1; timezone=Europe%2FMoscow; language=ru; language-data=ru%2Cen; currency=rub; user=c4ca4238a0b923820dcc509a6f75849b\n\ntest")
	request := http.Request{}
	request.Parse(buff)
	fmt.Println(request.Method)
	fmt.Println(request.Protocol)
	fmt.Println(request.Version)
	fmt.Println(request.Iri)
	fmt.Println(request.Path)
	for _, param := range request.Params {
		fmt.Printf("Param Name: \"%s\" = Value: \"%s\"\n", param.Name, param.Value)
	}
	for _, header := range request.Headers {
		fmt.Printf("Header Name: \"%s\" = Value: \"%s\"\n", header.Name, header.Value)
	}
	for _, cookie := range request.Cookies {
		fmt.Printf("Cookie Name: \"%s\" = Value: \"%s\"\n", cookie.Name, cookie.Value)
	}
	fmt.Println(string(request.Body))
}

func testResponse() {
	response := http.Response{
		Protocol: "HTTP",
		Version: "1.1",
		Code: 200,
		Status: "OK",
		Headers: []http.Header {
			{Name: "Server", Value: "goHttpServer/0.0.1"},
		},
		Body: []byte("<html><head><title>goHttpServer works!</title></head><body><h1>goHttpServer works!</h1></body></html>"),
	}
	fmt.Println(string(response.Build()))
}

func testFile()  {
	home := "./www"
	path := "/"

	reader := file.File{}
	indexes := []string{"", "index.html", "index.htm"}
	for _, index := range indexes {
		fileLocation := home + path
		if fileLocation[len(fileLocation)-1:] != "/" && index != "" {
			fileLocation = fileLocation + "/"
		}
		fileLocation = fileLocation + index
		reader.Info(fileLocation)
		fmt.Println("Name", reader.Name)
		fmt.Println("Exist", reader.Exist)
		if reader.Exist {
			fmt.Println("Directory", reader.Directory)
			if reader.Directory {
				continue
			}
			fmt.Println("Readable", reader.Readable)
			if reader.Readable {
				fmt.Println("Size", reader.Size)
				fmt.Println("Mimetype", reader.Mimetype)
				geoLocation, _ := time.LoadLocation("GMT")
				fmt.Println("DateTime", reader.DateTime.In(geoLocation).Format(time.RFC1123))
				fmt.Println(string(reader.Read()))
				break
			}
		}
	}
}
