package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"http-proxy/pkg/http_utils"
	"http-proxy/repo"
)

type Handler struct {
	certs         map[string][]byte
	mutex         sync.Mutex
	key           []byte
	requestSaver  repo.RequestSaver
	responseSaver repo.ResponseSaver
}

func NewHandler(req repo.RequestSaver, resp repo.ResponseSaver) (*Handler, error) {
	keyBytes, err := os.ReadFile("https/cert.key")
	if err != nil {
		return nil, err
	}

	certs, err := loadCertificates()
	if err != nil {
		return nil, err
	}

	return &Handler{
		certs:         certs,
		key:           keyBytes,
		requestSaver:  req,
		responseSaver: resp,
	}, nil
}

func (h *Handler) Handle(conn net.Conn) error {
	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		return err
	}

	return h.handleRequest(conn, req)
}

func (h *Handler) handleRequest(clientConn net.Conn, toProxy *http.Request) error {
	var hostConn net.Conn
	var err error

	host := toProxy.URL.Hostname()
	port := utils.GetPort(toProxy.URL)

	if toProxy.Method == http.MethodConnect {
		clientConn, err = h.tlsUpgrade(clientConn, host)
		if err != nil {
			return err
		}

		toProxy, err = http.ReadRequest(bufio.NewReader(clientConn))
		if err != nil {
			return err
		}

		toProxy.URL.Scheme = "https"
		hostConn, err = utils.TLSConnect(host, port)
		if err != nil {
			return err
		}
	} else {
		toProxy.URL.Scheme = "http"
		hostConn, err = utils.TCPConnect(host, port)
		if err != nil {
			return err
		}
	}

	defer hostConn.Close()

	prepareRequest(toProxy)
	requestId, err := h.requestSaver.Save(toProxy)
	if err != nil {
		return err
	}

	resp, err := utils.SendRequest(hostConn, toProxy)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	h.responseSaver.Save(requestId, resp)

	return utils.WriteResponse(resp, clientConn)
}

func (h *Handler) getTlsConfig(host string) (*tls.Config, error) {
	cert, err := tls.X509KeyPair(h.certs[host], h.key)
	if err != nil {
		return nil, err
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}

func (h *Handler) tlsUpgrade(clientConn net.Conn, host string) (net.Conn, error) {
	_, err := clientConn.Write([]byte("HTTP/1.0 200 Connection established\n\n"))
	if err != nil {
		return nil, err
	}

	err = h.generateCertificate(host)
	if err != nil {
		return nil, err
	}

	cfg, err := h.getTlsConfig(host)
	if err != nil {
		return nil, err
	}

	tlsConn := tls.Server(clientConn, cfg)
	clientConn.SetReadDeadline(time.Now().Add(utils.DefaultTimeout))

	return tlsConn, nil
}

func (h *Handler) generateCertificate(host string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	_, exists := h.certs[host]
	if !exists {
		fmt.Printf("Generating certificate for %s\n", host)
		cert, err := generateCertificate(host)
		if err != nil {
			return fmt.Errorf("error generating certificate: %v", err)
		}
		h.certs[host] = cert
	}

	return nil
}

func prepareRequest(r *http.Request) {
	r.URL.Host = ""
	r.Header.Del("Proxy-Connection")
	r.Header.Del("Accept-Encoding")
}

func loadCertificates() (map[string][]byte, error) {
	entries, err := os.ReadDir("certs")
	if err != nil {
		return nil, err
	}

	res := make(map[string][]byte, len(entries))

	for _, entry := range entries {
		host := strings.TrimSuffix(entry.Name(), ".crt")

		res[host], err = os.ReadFile("certs/" + entry.Name())
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func generateCertificate(host string) ([]byte, error) {
	cmd := exec.Command("./https/gen.sh", host)
	var out strings.Builder
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	res := []byte(out.String())

	cert, err := os.Create(fmt.Sprintf("certs/%s.crt", host))
	if err != nil {
		return nil, err
	}

	defer cert.Close()

	var written int64 = 0

	written, err = io.Copy(cert, bytes.NewReader(res))
	if err != nil {
		return nil, err
	}

	if written == 0 {
		return nil, errors.New("0 bytes written during certificate creation")
	}

	return res, nil
}
