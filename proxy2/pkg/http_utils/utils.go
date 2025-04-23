package utils

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const (
	defaultHTTPPort  = "80"
	defaultHTTPSPort = "443"
	DefaultTimeout   = 10 * time.Second
)

func WriteError(err error, conn net.Conn) error {
	resp := &http.Response{
		Status:     http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("Proxy error: " + err.Error() + "\n")),
	}
	resp.Header.Set("Content-Type", "text/plain")

	return WriteResponse(resp, conn)
}

func HTTPError(w http.ResponseWriter, message string, statusCode int, err error) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(statusCode)

	if message != "" {
		w.Write([]byte(message + ": "))
	}
	if err != nil {
		w.Write([]byte(err.Error() + "\n"))
	}
}

func DumpRequest(r *http.Request, includeBody bool) string {
	dump, err := httputil.DumpRequest(r, includeBody)
	if err != nil {
		return fmt.Sprintf("Error dumping request: %v", err)
	}
	return string(dump)
}

func DumpResponse(r *http.Response, includeBody bool) string {
	dump, err := httputil.DumpResponse(r, includeBody)
	if err != nil {
		return fmt.Sprintf("Error dumping response: %v", err)
	}
	return string(dump)
}

func WriteResponse(resp *http.Response, conn net.Conn) error {
	if err := conn.SetWriteDeadline(time.Now().Add(DefaultTimeout)); err != nil {
		return fmt.Errorf("set write deadline failed: %w", err)
	}

	bytes, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return fmt.Errorf("dump response failed: %w", err)
	}

	if _, err := conn.Write(bytes); err != nil {
		return fmt.Errorf("write response failed: %w", err)
	}
	return nil
}

func GetPort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}

	switch strings.ToLower(u.Scheme) {
	case "https":
		return defaultHTTPSPort
	default:
		return defaultHTTPPort
	}
}

func SendRequest(conn net.Conn, req *http.Request) (*http.Response, error) {
	if err := conn.SetWriteDeadline(time.Now().Add(DefaultTimeout)); err != nil {
		return nil, fmt.Errorf("set write deadline failed: %w", err)
	}

	bytes, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, fmt.Errorf("dump request failed: %w", err)
	}

	if _, err := conn.Write(bytes); err != nil {
		return nil, fmt.Errorf("write request failed: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(DefaultTimeout)); err != nil {
		return nil, fmt.Errorf("set read deadline failed: %w", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	return resp, nil
}

func TCPConnect(host, port string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout: DefaultTimeout,
	}

	conn, err := dialer.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, fmt.Errorf("TCP connect failed: %w", err)
	}
	return conn, nil
}

func TLSConnect(host, port string) (net.Conn, error) {
	dialer := &tls.Dialer{
		Config: &tls.Config{},
		NetDialer: &net.Dialer{
			Timeout: DefaultTimeout,
		},
	}

	conn, err := dialer.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, fmt.Errorf("TLS connect failed: %w", err)
	}
	return conn, nil
}

func CopyData(dst, src net.Conn) error {
	if err := dst.SetDeadline(time.Now().Add(DefaultTimeout)); err != nil {
		return fmt.Errorf("set deadline on dst failed: %w", err)
	}
	if err := src.SetDeadline(time.Now().Add(DefaultTimeout)); err != nil {
		return fmt.Errorf("set deadline on src failed: %w", err)
	}

	_, err := io.Copy(dst, src)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("copy data failed: %w", err)
	}
	return nil
}
