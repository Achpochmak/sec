package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
)

var (
	requests     []*http.Request
	requestsLock sync.Mutex
)

func handleProxyRequest(clientConn net.Conn) {
	defer clientConn.Close()

	req, err := http.ReadRequest(bufio.NewReader(clientConn))
	if err != nil {
		log.Println("Error reading request:", err)
		return
	}

	requestsLock.Lock()
	requests = append(requests, req)
	reqID := len(requests) - 1
	requestsLock.Unlock()

	log.Printf("Received request ID: %d, URL: %s\n", reqID, req.URL.String())

	if req.Method == http.MethodConnect {
		handleTunneling(clientConn, req)
		return
	}

	targetURL := req.URL
	targetURL.Host = "webapi"
	if targetURL.Scheme == "" {
		targetURL, err = url.Parse(req.RequestURI)
		if err != nil {
			log.Println("Error parsing URL:", err)
			return
		}
	}

	log.Printf("Target URL: %s", targetURL.String())

	targetConn, err := net.Dial("tcp", targetURL.Host+":8000")
	if err != nil {
		log.Println("Error connecting to target:", err)
		return
	}
	defer targetConn.Close()

	req.RequestURI = ""
	req.Header.Del("Proxy-Connection")

	if err := req.Write(targetConn); err != nil {
		log.Println("Error writing to target:", err)
		return
	}

	res, err := http.ReadResponse(bufio.NewReader(targetConn), req)
	if err != nil {
		log.Println("Error reading response from target:", err)
		return
	}
	defer res.Body.Close()

	if err := res.Write(clientConn); err != nil {
		log.Println("Error writing response to client:", err)
		return
	}

	log.Printf("Proxied request ID: %d, Status: %s\n", reqID, res.Status)
}

func handleTunneling(clientConn net.Conn, req *http.Request) {
	targetConn, err := net.Dial("tcp", req.URL.Host)
	if err != nil {
		log.Println("Error dialing target:", err)
		return
	}
	defer targetConn.Close()

	_, err = fmt.Fprint(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")
	if err != nil {
		log.Println("Error writing response:", err)
		return
	}

	go transfer(clientConn, targetConn)
	go transfer(targetConn, clientConn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()

	buf := make([]byte, 4096)
	for {
		n, err := source.Read(buf)
		if err != nil {
			return
		}
		_, err = destination.Write(buf[:n])
		if err != nil {
			return
		}
	}
}

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error listening on port 8080: %v", err)
		os.Exit(1)
	}
	defer listener.Close()

	log.Println("Proxy server listening on :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go handleProxyRequest(conn)
	}
}
