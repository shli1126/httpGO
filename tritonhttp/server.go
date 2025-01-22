package tritonhttp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	responseProto    = "HTTP/1.1"
	statusOK         = 200
	statusBadRequest = 400
	statusNotFound   = 404
)

var statusText = map[int]string{
	statusOK:         "OK",
	statusBadRequest: "Bad Request",
	statusNotFound:   "Not Found",
}

type Server struct {
	// Addr specifies the TCP address for the server to listen on,
	// in the form "host:port". It shall be passed to net.Listen()
	// during ListenAndServe().
	Addr string // e.g. ":0"

	// VirtualHosts contains a mapping from host name to the docRoot path
	// (i.e. the path to the directory to serve static files from) for
	// all virtual hosts that this server supports
	VirtualHosts map[string]string
}

func (s *Server) ListenAndServe() error {
	// Validate the configuration of the server
	if err := s.ValidateServerSetup(); err != nil {
		return fmt.Errorf("server is not setup correctly %v", err)
	}
	fmt.Println("Server setup valid!")

	// server should now start to listen on the configured address
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	fmt.Println("Listening on", ln.Addr())

	// making sure the listener is closed when we exit
	defer func() {
		err = ln.Close()
		if err != nil {
			fmt.Println("error in closing listener", err)
		}
	}()

	// accept connections forever
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		fmt.Println("accepted connection", conn.RemoteAddr())
		go s.HandleConnection(conn)
	}
}

// readline give error when nothing to read?
func ReadLine(br *bufio.Reader) (string, error) {
	var line string
	for {
		s, err := br.ReadString('\n')
		line += s
		// Return the error
		if err != nil {
			return line, err
		}
		// Return the line when reaching line end
		if strings.HasSuffix(line, "\r\n") {
			// Striping the line end
			line = line[:len(line)-2]
			return line, nil
		}
	}
}

func ReadRequest(br *bufio.Reader) (req *Request, err error) {
	// Read start line
	line, err := ReadLine(br)
	if err != nil {
		return nil, errors.New("read null request")
	}
	req = &Request{}
	req.Method, req.URL, req.Proto, err = parseRequestLine(line)
	if err != nil {
		return nil, badStringError("malformed request line", line)
	}
	// check if method is get
	if !validMethod(req.Method) {
		return nil, badStringError("invalid method", req.Method)
	}
	// check if start with /
	if !validURL(req.URL) {
		return nil, badStringError("invalid url", req.URL)
	}
	// check if HTTP1.1
	if !validProto(req.Proto) {
		return nil, badStringError("invalid proto", req.Proto)
	}
	// Read other lines of requests
	req.Headers = make(map[string]string)
	for {
		line, err := ReadLine(br)
		if err != nil {
			fmt.Println("what is the error then: ", err)
			return nil, err
		}
		fmt.Println("Line : ", line)
		if line == "" {
			// This marks header end
			break
		}
		key, val, err := parseHeaderLine(line)
		if err != nil {
			return nil, badStringError("malformed header line", line)
		}
		req.Headers[CanonicalHeaderKey(key)] = formatHeaderVal(val)
		fmt.Println("Read line from request", line)
	}
	//check if contain Host
	val, ok := req.Headers["Host"]
	if ok {
		req.Host = val
	} else {
		return nil, errors.New("request header does not contain host, 400 client error")
	}
	val, ok = req.Headers["Connection"]
	if ok {
		if val == "close" {
			req.Close = true
		} else {
			req.Close = false
		}
	}
	return req, nil
}

// HandleConnection reads requests from the accepted conn and handles them.
func (s *Server) HandleConnection(conn net.Conn) {
	br := bufio.NewReader(conn)
	// continuously read from  connection
	for {
		// Set timeout
		if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			log.Println(err)
			log.Printf("Failed to set timeout for connection %v becasue fail to set timeout", conn)
			_ = conn.Close()
			return
		}
		// Read next request from the client
		req, err := ReadRequest(br)
		fmt.Println("timeout readrequest error is what?????", err)
		// Handle EOF
		if errors.Is(err, io.EOF) {
			log.Printf("Connection closed by %v because of end of file", conn.RemoteAddr())
			_ = conn.Close()
			return
		}
		if err, ok := err.(net.Error); ok && err.Timeout() {
			//if haven't finish sending return 400
			if err.Error() == ("read null request") {
				log.Printf("timeout occur and haven't receive new request, close right away")
				_ = conn.Close()
				return
			} else {
				//partial request when time out
				fmt.Println("enter timeout")
				res := &Response{}
				res.HandleBadRequest(req)
				fmt.Println("Enter the time out Write, the following flush will flush the timeout")
				err := res.Write(conn, req)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println("Connection close because of timeout")
				_ = conn.Close()
				return
			}
		}
		// read request has error
		if err != nil {
			if err.Error() != ("read null request") {
				log.Printf("Handle bad request for error: %v", err)
				res := &Response{}
				res.HandleBadRequest(req)
				fmt.Println("Enter the bad request Write, the following flush will flush the bad request")
				err = res.Write(conn, req)
				if err != nil {
					fmt.Println(err)
					_ = conn.Close()
					return
				}
				_ = conn.Close()
				return
			}
			fmt.Println("what is the error here?????", err)
			_ = conn.Close()
			return
			// fmt.Println("Having a null request")
			// _ = conn.Close()
		}

		// Handle good request
		log.Printf("Handle good request: %v", req)
		res := s.HandleGoodRequest(req)
		fmt.Println("Enter the good request Write, the following flush will flush the good request")
		err = res.Write(conn, req)
		if err != nil {
			fmt.Println(err)
			return
		}
		if req.Close {
			log.Printf("Connection close because the request ask me to do so")
			_ = conn.Close()
			return
		}
		// We'll never close the connection and handle as many requests for this connection and pass on this
		// responsibility to the timeout mechanism
	}
}

func getSortedKey(headers map[string]string) []string {
	sortedKeys := make([]string, 0, len(headers))
	for k := range headers {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)
	return sortedKeys
}

func (res *Response) Write(conn net.Conn, req *Request) error {
	bw := bufio.NewWriter(conn)
	statusLine := fmt.Sprintf("%v %v %v\r\n", res.Proto, res.StatusCode, statusText[res.StatusCode])
	if _, err := bw.WriteString(statusLine); err != nil {
		fmt.Println("having error while writing the status line")
		return err
	}
	if err := bw.Flush(); err != nil {
		fmt.Println("what is the statusline in flush:", statusLine)
		fmt.Println("haveing error while flushing the status line", err)
		return err
	}
	fmt.Println("reach here?")
	sortedKeys := getSortedKey(res.Headers)
	for _, k := range sortedKeys {
		v := res.Headers[k]
		headerLine := fmt.Sprintf("%v: %v\r\n", CanonicalHeaderKey(k), v)
		if _, err := bw.WriteString(headerLine); err != nil {
			return err
		}
		if err := bw.Flush(); err != nil {
			return err
		}
	}
	if _, err := bw.WriteString("\r\n"); err != nil {
		return err
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	fmt.Println("ever reach here???????????")
	if res.StatusCode == 200 {
		file, err := os.ReadFile(res.FilePath)
		if err != nil {
			return err
		}
		conn.Write(file)
	}
	return nil
}

// HandleBadRequest prepares res to be a 405 Method Not allowed response
func (res *Response) HandleBadRequest(req *Request) {
	res.Proto = responseProto
	res.StatusCode = statusBadRequest
	res.StatusText = statusText[400]
	res.Request = req
	res.FilePath = ""
	res.Headers = make(map[string]string)
	res.Headers["Date"] = FormatTime(time.Now())
	res.Headers["Connection"] = "close"
}

// 404 in handle good request
func (s *Server) HandleGoodRequest(req *Request) (r *Response) {
	filePath := s.formatFilePath(req.Host, req.URL)
	fmt.Println("formated file path is:", filePath)
	exist, err := fileExist(filePath)
	//handle file not exist
	if err != nil || !exist {
		fmt.Println("error occur while locating file:", err)
		return s.HandleGoodNotFoundRequest(req)
	}
	fmt.Println("I should have got this filepath:", filePath)
	//handle file exist
	r = &Response{}
	r.Proto = responseProto
	r.StatusCode = statusOK
	r.StatusText = statusText[r.StatusCode]
	r.Request = req
	r.FilePath = filePath
	lastModified, contentLength, contentType, err := getFileInfo(filePath)
	if err != nil {
		fmt.Println("cannot get file info", err)
		return
	}
	r.Headers = make(map[string]string)
	r.Headers["Date"] = FormatTime(time.Now())
	r.Headers["Last-Modified"] = lastModified
	r.Headers["Content-Type"] = contentType
	r.Headers["Content-Length"] = contentLength
	val, ok := req.Headers["Connection"]
	if ok && req.Close {
		r.Headers["Connection"] = val
	}
	return r
}

func (s *Server) HandleGoodNotFoundRequest(req *Request) (r *Response) {
	r = &Response{}
	r.Proto = responseProto
	r.StatusCode = statusNotFound
	r.StatusText = statusText[r.StatusCode]
	r.Request = req
	r.FilePath = ""
	r.Headers = make(map[string]string)
	r.Headers["Date"] = FormatTime(time.Now())
	val, ok := req.Headers["Connection"]
	if ok && req.Close{
		r.Headers["Connection"] = val
	}
	return r
}

// get Last Modified, Content-Length, Content-Type,
func getFileInfo(filePath string) (string, string, string, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", "", "", fmt.Errorf("could not get file info")
	}
	modificationTime := FormatTime(fileInfo.ModTime())
	fileSize := strconv.FormatInt(fileInfo.Size(), 10)
	fileType := MIMETypeByExtension(filepath.Ext(filePath))
	fileType = strings.SplitN(fileType, ";", 2)[0]
	return modificationTime, fileSize, fileType, nil
}

func fileExist(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// relative to absolute
func (s *Server) formatFilePath(host string, relativeURL string) string {

	if strings.HasSuffix(relativeURL, "/") {
		relativeURL = filepath.Join(relativeURL, "index.html")
	}
	absoluteURL := filepath.Join(s.VirtualHosts[host], relativeURL)
	absoluteURL = filepath.Clean(absoluteURL)
	fmt.Println("The absolute path is:", absoluteURL)
	return absoluteURL
}

// parseRequestLine parses "GET /foo HTTP/1.1" into its individual parts.
func parseRequestLine(line string) (string, string, string, error) {
	fields := strings.SplitN(line, " ", 3)
	if len(fields) != 3 {
		return "", "", "", fmt.Errorf("could not parse the request line, got fields %v", fields)
	}
	return fields[0], fields[1], fields[2], nil
}

func checkValidKey(key string) bool {
	for _, v := range key {
		if !unicode.IsLetter(v) && !unicode.IsDigit(v) && v != '-' {
			return false
		}
	}
	return true
}

func parseHeaderLine(line string) (string, string, error) {
	fields := strings.SplitN(line, ":", 2)
	if len(fields) != 2 {
		return "", "", fmt.Errorf("could not parse the request line, missing colon, got fields %v", fields)
	}
	if fields[0] == "" {
		return "", "", fmt.Errorf("key cannot be empty")
	}
	if !checkValidKey(fields[0]) {
		return "", "", fmt.Errorf("key not in correct format")
	}
	return fields[0], fields[1], nil
}

func formatHeaderVal(val string) string {
	return strings.TrimLeftFunc(val, unicode.IsSpace)
}

func validMethod(method string) bool {
	return method == "GET"
}

func validURL(url string) bool {
	return strings.HasPrefix(url, "/")
}

func validProto(proto string) bool {
	return proto == "HTTP/1.1"
}
func badStringError(what, val string) error {
	return fmt.Errorf("%s %q", what, val)
}

func (s *Server) ValidateServerSetup() error {
	for host, docRoot := range s.VirtualHosts {
		// Validating the doc root of the server
		fmt.Println("Host is:", host)
		fmt.Println("docRoot is:", docRoot)
		fi, err := os.Stat(docRoot)
		if os.IsNotExist(err) {
			return err
		}
		if !fi.IsDir() {
			return fmt.Errorf("doc root %q is not a directory", docRoot)
		}
	}
	return nil
}
