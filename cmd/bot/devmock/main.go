package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

type mockExpectation struct {
	HTTPRequest struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	} `json:"httpRequest"`
	HTTPResponse struct {
		StatusCode int                 `json:"statusCode"`
		Headers    map[string][]string `json:"headers"`
		Body       string              `json:"body"`
	} `json:"httpResponse"`
}

type mockResponse struct {
	statusCode int
	headers    map[string][]string
	body       string
}

func main() {
	addr := flag.String("addr", ":1080", "HTTP listen address")
	configPath := flag.String("config", "mock-onec-config.json", "path to MockServer-compatible JSON expectations")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	routes, err := loadRoutes(*configPath)
	if err != nil {
		log.Error("load mock config", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		key := routeKey(r.Method, r.URL.Path)
		resp, ok := routes[key]
		if !ok {
			log.Warn("mock route not found", "method", r.Method, "path", r.URL.Path)
			writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "message": "mock route not found"})
			return
		}

		for name, values := range resp.headers {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(resp.statusCode)
		_, _ = w.Write([]byte(resp.body))
		log.Info("mock response", "method", r.Method, "path", r.URL.Path, "status", resp.statusCode)
	})

	server := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Info("dev mock server started", "addr", *addr, "config", *configPath, "routes", len(routes))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("dev mock server stopped", "err", err)
		os.Exit(1)
	}
}

func loadRoutes(path string) (map[string]mockResponse, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	var expectations []mockExpectation
	if err := json.NewDecoder(file).Decode(&expectations); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}

	routes := make(map[string]mockResponse, len(expectations))
	for _, expectation := range expectations {
		method := strings.TrimSpace(expectation.HTTPRequest.Method)
		path := strings.TrimSpace(expectation.HTTPRequest.Path)
		if method == "" || path == "" {
			return nil, fmt.Errorf("mock expectation has empty method or path")
		}
		statusCode := expectation.HTTPResponse.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		routes[routeKey(method, path)] = mockResponse{
			statusCode: statusCode,
			headers:    expectation.HTTPResponse.Headers,
			body:       expectation.HTTPResponse.Body,
		}
	}
	return routes, nil
}

func routeKey(method, path string) string {
	return strings.ToUpper(method) + " " + path
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
