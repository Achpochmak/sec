package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

var (
	requests []http.Request
)

type RequestDetails struct {
	ID      int
	Method  string
	URL     string
	Headers map[string][]string
}

func getRequestsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requests)
}

func getRequestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= len(requests) {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	req := requests[id]
	reqDetails := RequestDetails{
		ID:      id,
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: req.Header,
	}

	json.NewEncoder(w).Encode(reqDetails)
}

func repeatRequestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= len(requests) {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	req := requests[id]
	// Copy the request (necessary to avoid modifying the original)
	newReq, err := http.NewRequest(req.Method, req.URL.String(), req.Body)
	if err != nil {
		http.Error(w, "Error creating new request", http.StatusInternalServerError)
		return
	}
	newReq.Header = req.Header

	client := &http.Client{}
	resp, err := client.Do(newReq)
	if err != nil {
		http.Error(w, "Error sending request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Error copying response body: %v", err)
	}
}

func scanRequestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= len(requests) {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	req := requests[id]
	vulnerabilities := performVulnerabilityScan(&req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vulnerabilities)
}

func performVulnerabilityScan(req *http.Request) []string {
	vulnerabilities := []string{}
	if req.Method == http.MethodGet && strings.Contains(req.URL.String(), "admin=true") {
		vulnerabilities = append(vulnerabilities, "Potential admin access vulnerability")
	}
	if req.Header.Get("X-Custom-Header") == "secret" {
		vulnerabilities = append(vulnerabilities, "Custom header 'X-Custom-Header' contains a sensitive value")
	}
	return vulnerabilities
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/requests", getRequestsHandler)
	r.HandleFunc("/requests/{id}", getRequestHandler)
	r.HandleFunc("/repeat/{id}", repeatRequestHandler)
	r.HandleFunc("/scan/{id}", scanRequestHandler)
	http.Handle("/", r)

	log.Println("Web API listening on :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
