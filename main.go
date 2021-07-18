package main

import (
	"fmt"
	"net"
)

func main() {
	// http methods:
	//OPTIONS
	//GET
	//HEAD
	//POST
	//PUT
	//PATCH
	//DELETE
	//TRACE
	//CONNECT
	// https://alexeykalina.github.io/technologies/sockets-go.html
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
	//GET /radio/listen/ HTTP/1.1
	//Host: artem-aleksashkin
	//Connection: keep-alive
	//Cache-Control: max-age=0
	//DNT: 1
	//Upgrade-Insecure-Requests: 1
	//User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.182 Safari/537.36
	//Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9
	//Referer: http://artem-aleksashkin/
	//Accept-Encoding: gzip, deflate
	//Accept-Language: ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7
	//Cookie: id=ca2a568d-cdf0-4658-b98c-320a9f1b0eb6; geography=1; timezone=Europe%2FMoscow; language=ru; language-data=ru%2Cen; currency=rub; user=c4ca4238a0b923820dcc509a6f75849b

	//HTTP/1.1 200 OK
	//Server: nginx/1.14.0 (Ubuntu)
	//Date: Sun, 18 Jul 2021 01:13:24 GMT
	//Content-Type: text/html;charset=UTF-8
	//Transfer-Encoding: chunked
	//Connection: keep-alive
	//Set-Cookie: geography=1; expires=Sun, 01-Aug-2021 01:13:24 GMT; Max-Age=1209600; path=/; domain=artem-aleksashkin
	//Content-Encoding: gzip

	defer conn.Close() // закрываем сокет при выходе из функции

	//buf := make([]byte, 32) // буфер для чтения клиентских данных
	for {
		//readLen, err := conn.Read(buf) // читаем из сокета
		//if err != nil {
		//	fmt.Println(err)
		//	break
		//}
		//

		conn.Write([]byte("HTTP/1.1 200 OK\n"))
		conn.Write([]byte("Server: goHttpServer\n"))
		conn.Write([]byte("\n"))
		conn.Write([]byte("<html><head><title>goHttpServer works!</title></head><body><h1>goHttpServer works!</h1></body></html>"))
		break
	}
}
