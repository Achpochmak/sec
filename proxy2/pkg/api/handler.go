package api

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"time"

	"http-proxy/pkg/http_utils"
	"http-proxy/pkg/xxe"
	"http-proxy/repo"

	"github.com/gorilla/mux"
)

const (
	defaultListSize     = 5
	defaultTimeout      = 30 * time.Second
	tlsHandshakeTimeout = 5 * time.Second
)

type Handler struct {
	requests  repo.RequestSaver
	responses repo.ResponseSaver
	client    *http.Client
}

func NewHandler(req repo.RequestSaver, resp repo.ResponseSaver) (*Handler, error) {
	transport, err := createSecureTransport()
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	return &Handler{
		requests:  req,
		responses: resp,
		client: &http.Client{
			Transport:     transport,
			Timeout:       defaultTimeout,
			CheckRedirect: noRedirectPolicy,
		},
	}, nil
}

func noRedirectPolicy(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func createSecureTransport() (*http.Transport, error) {
	config, err := createTLSConfig()
	if err != nil {
		return nil, err
	}

	return &http.Transport{
		TLSClientConfig:     config,
		TLSHandshakeTimeout: tlsHandshakeTimeout,
	}, nil
}

func createTLSConfig() (*tls.Config, error) {
	cert, err := os.ReadFile("https/ca.crt")
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(cert) {
		return nil, errors.New("failed to append CA certificate")
	}

	return &tls.Config{
		RootCAs: pool,
	}, nil
}

func (h *Handler) GetRequest(w http.ResponseWriter, r *http.Request) {
	requestID := mux.Vars(r)["id"]
	req, err := h.requests.Get(requestID)
	if err != nil {
		utils.HTTPError(w, "Failed to get request", http.StatusNotFound, err)
		return
	}

	if err := encodeJSONResponse(w, req); err != nil {
		utils.HTTPError(w, "Failed to encode response", http.StatusInternalServerError, err)
	}
}

func (h *Handler) ListRequests(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimitParam(r)
	if err != nil {
		limit = defaultListSize
	}

	requests, err := h.requests.List(limit)
	if err != nil {
		utils.HTTPError(w, "Failed to list requests", http.StatusInternalServerError, err)
		return
	}

	if err := encodeJSONResponse(w, requests); err != nil {
		utils.HTTPError(w, "Failed to encode response", http.StatusInternalServerError, err)
	}
}

func parseLimitParam(r *http.Request) (int64, error) {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		return defaultListSize, nil
	}
	return strconv.ParseInt(limitStr, 10, 64)
}

func (h *Handler) RepeatRequest(w http.ResponseWriter, r *http.Request) {
	requestID := mux.Vars(r)["id"]
	req, err := h.requests.GetEncoded(requestID)
	if err != nil {
		utils.HTTPError(w, "Failed to get request", http.StatusNotFound, err)
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		utils.HTTPError(w, "Failed to repeat request", http.StatusBadGateway, err)
		return
	}
	defer resp.Body.Close()

	if err := forwardResponse(w, resp); err != nil {
		utils.HTTPError(w, "Failed to forward response", http.StatusInternalServerError, err)
	}
}

func forwardResponse(w http.ResponseWriter, resp *http.Response) error {
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	_, err := io.Copy(w, resp.Body)
	return err
}

func (h *Handler) DumpRequest(w http.ResponseWriter, r *http.Request) {
	requestID := mux.Vars(r)["id"]
	req, err := h.requests.GetEncoded(requestID)
	if err != nil {
		utils.HTTPError(w, "Failed to get request", http.StatusNotFound, err)
		return
	}

	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		utils.HTTPError(w, "Failed to dump request", http.StatusInternalServerError, err)
		return
	}

	w.Write(dump)
}

func (h *Handler) ScanRequest(w http.ResponseWriter, r *http.Request) {
	requestID := mux.Vars(r)["id"]
	req, err := h.requests.GetEncoded(requestID)
	if err != nil {
		utils.HTTPError(w, "Failed to get request", http.StatusNotFound, err)
		return
	}

	hadXML, err := xxe.AddVulnerability(req)
	if err != nil {
		utils.HTTPError(w, "Failed to modify request", http.StatusInternalServerError, err)
		return
	}

	if !hadXML {
		w.Write([]byte("No XML content in request\n"))
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		utils.HTTPError(w, "Failed to scan request", http.StatusBadGateway, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.HTTPError(w, "Failed to read response", http.StatusInternalServerError, err)
		return
	}

	result := buildScanResult(body)
	w.Write(result)
}

func buildScanResult(body []byte) []byte {
	if xxe.IsVulnerable(body) {
		return []byte("Request vulnerable, response:\n" + string(body) + "\n")
	}
	return []byte("Request is not vulnerable, response:\n" + string(body) + "\n")
}

func (h *Handler) GetResponse(w http.ResponseWriter, r *http.Request) {
	responseID := mux.Vars(r)["id"]
	resp, err := h.responses.Get(responseID)
	if err != nil {
		utils.HTTPError(w, "Failed to get response", http.StatusNotFound, err)
		return
	}

	if err := encodeJSONResponse(w, resp); err != nil {
		utils.HTTPError(w, "Failed to encode response", http.StatusInternalServerError, err)
	}
}

func (h *Handler) GetRequestResponse(w http.ResponseWriter, r *http.Request) {
	requestID := mux.Vars(r)["id"]
	resp, err := h.responses.GetByRequest(requestID)
	if err != nil {
		utils.HTTPError(w, "Failed to get response", http.StatusNotFound, err)
		return
	}

	if err := encodeJSONResponse(w, resp); err != nil {
		utils.HTTPError(w, "Failed to encode response", http.StatusInternalServerError, err)
	}
}

func (h *Handler) ListResponses(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimitParam(r)
	if err != nil {
		limit = defaultListSize
	}

	responses, err := h.responses.List(limit)
	if err != nil {
		utils.HTTPError(w, "Failed to list responses", http.StatusInternalServerError, err)
		return
	}

	if err := encodeJSONResponse(w, responses); err != nil {
		utils.HTTPError(w, "Failed to encode response", http.StatusInternalServerError, err)
	}
}

func encodeJSONResponse(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(data)
}
