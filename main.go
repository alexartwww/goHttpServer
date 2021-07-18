package main

import (
	"alexartwww/goHttpServer/http"
	"fmt"
	"io"
	"net"
)

func main() {
	fmt.Println("Listen http://localhost:4321")
	listener, _ := net.Listen("tcp", "localhost:4321") // открываем слушающий сокет
	for {
		conn, err := listener.Accept() // принимаем TCP-соединение от клиента и создаем новый сокет
		if err != nil {
			continue
		}
		go handleClient(conn) // обрабатываем запросы клиента в отдельной го-рутине
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close() // закрываем сокет при выходе из функции

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

	// read files

	response := http.Response{
		Protocol: request.Protocol,
		Version: request.Version,
		Code: 200,
		Status: "OK",
		Headers: []http.Header {
			{Name: "Server", Value: "goHttpServer/0.0.1"},
		},
		Body: []byte("<html><head><title>goHttpServer works!</title></head><body><h1>goHttpServer works!</h1></body></html>"),
	}
	conn.Write(response.Build())
}

// =======================================================================

func testRequest() {
	buff := []byte("GET /radio/listen/ HTTP/1.1\nHost: artem-aleksashkin\nConnection: keep-alive\nCache-Control: max-age=0\nDNT: 1\nUpgrade-Insecure-Requests: 1\nUser-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.182 Safari/537.36\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9\nReferer: http://artem-aleksashkin/\nAccept-Encoding: gzip, deflate\nAccept-Language: ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7\nCookie: id=ca2a568d-cdf0-4658-b98c-320a9f1b0eb6; geography=1; timezone=Europe%2FMoscow; language=ru; language-data=ru%2Cen; currency=rub; user=c4ca4238a0b923820dcc509a6f75849b\n\ntest")
	request := http.Request{
		Method: "",
		Iri: "",
		Protocol: "",
		Version: "",
		Headers: make([]http.Header, 0),
		Cookies: make([]http.Cookie, 0),
		Body: make([]byte, 0)}
	request.Parse(buff)
	fmt.Println(request.Method)
	fmt.Println(request.Iri)
	fmt.Println(request.Protocol)
	fmt.Println(request.Version)
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
