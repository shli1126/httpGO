package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	var conn net.Conn
	var err error
	for {
		time.Sleep(250 * time.Millisecond)
		conn, err = net.Dial("tcp", "localhost:8080")
		if err == nil {
			break
		}
	}
	defer conn.Close()

	// follow the protocal
	test := make([]byte, 10)
	size, err := conn.Read(test)
	fmt.Print("Response is: ", test[:size])
}

// buffer := make([]byte, 1000)
// req := ""
// for {
// 	size, err := conn.Read(buffer)
// 	if err != nil {
// 		if err != io.EOF {
// 			fmt.Println("Read error", err)
// 		}
// 		break
// 	}
// 	data := buffer[:size]
// 	req = req + string(data)
// }
// //create the request object
// req = strings.TrimRight(req, "\r\n")
// request := strings.Split(req, "\n")
// //request[0] is initial line
// initial := handleInitialLine(request[0])
// method, url, proto := initial[0], initial[1], initial[2]
// //request[1:] is the headers
// header := handleHeader(request[1:])
// host, close := header["Host"], header["Connection"]
// formated_req := Request{
// 	Method:  method,
// 	URL:     url,
// 	Proto:   proto,
// 	Headers: header,
// 	Host:    host,
// 	Close:   close == "close",
// }
// //use write to return the response
// formated_res := Response{
// 	Proto: proto,
// 	StatusCode: 200,
// 	StatusText: "OK",
// 	Headers: header,
// 	Request: &formated_req,
// 	FilePath: "/",
// }
// initial_response_line := formated_res.Proto + " " + strconv.Itoa(formated_res.StatusCode) + " " + formated_res.StatusText + "\r\n"
// response_header := ""
// for k, v := range formated_res.Headers {
// 	response_header += k + ": " + v + "\r\n"
// }
// response_header += "\r\n"

// _, err := conn.Write([]byte(initial_response_line + response_header))
// if err != nil {
// 	fmt.Println(err)
// }

// func handleInitialLine(initial string) []string {
// 	// GET, <URL>, HTTP/1.1
// 	components := strings.Split(initial, " ")
// 	return components
// }

// func handleHeader(header []string) map[string]string {
// 	pairs := make(map[string]string)
// 	for _, v := range header {
// 		format := strings.TrimSuffix(v, "\n")
// 		pair := strings.Split(format, ": ")
// 		if len(pair) != 2 {
// 			fmt.Println("Invalid key value pair")
// 		}
// 		key, val := strings.TrimSpace(pair[0]), strings.TrimSpace(pair[1])
// 		key = CanonicalHeaderKey(key)
// 		pairs[key] = val
// 	}
// 	return pairs
// }
