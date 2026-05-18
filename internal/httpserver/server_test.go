package httpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	maxclient "example.com/max-bot-go/internal/clients/max"
	"example.com/max-bot-go/internal/clients/onec"
	"example.com/max-bot-go/internal/config"
	"example.com/max-bot-go/internal/service"
	"example.com/max-bot-go/internal/store"
)

func TestDebugSendTestUpdateRequiresInternalToken(t *testing.T) {
	srv := New(testConfig(true), slog.New(slog.NewTextHandler(io.Discard, nil)), nil)

	req := httptest.NewRequest(http.MethodPost, "/debug/send-test-update", bytes.NewBufferString(`{"user_id":1,"chat_id":2,"text":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d with body %q", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestDebugSendTestUpdateAcceptsValidInternalToken(t *testing.T) {
	processed := make(chan struct{}, 1)
	maxAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			t.Errorf("unexpected MAX path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "max-token" {
			t.Errorf("unexpected MAX Authorization header: %q", got)
		}
		processed <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer maxAPI.Close()

	oneCAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "operation_id": "op-test", "data": map[string]any{}})
	}))
	defer oneCAPI.Close()

	botService := service.New(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		store.NewMemoryStore(),
		maxclient.New(maxAPI.URL, "max-token", time.Second),
		onec.New(oneCAPI.URL, "onec-token", time.Second),
	)
	srv := New(testConfig(true), slog.New(slog.NewTextHandler(io.Discard, nil)), botService)

	req := httptest.NewRequest(http.MethodPost, "/debug/send-test-update", bytes.NewBufferString(`{"user_id":1,"chat_id":2,"text":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer internal-token")
	rec := httptest.NewRecorder()

	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %q", http.StatusOK, rec.Code, rec.Body.String())
	}

	select {
	case <-processed:
	case <-time.After(time.Second):
		t.Fatal("expected debug update to be processed")
	}
}

func TestDebugSendTestUpdateNotRegisteredWhenDisabled(t *testing.T) {
	srv := New(testConfig(false), slog.New(slog.NewTextHandler(io.Discard, nil)), nil)

	req := httptest.NewRequest(http.MethodPost, "/debug/send-test-update", bytes.NewBufferString(`{"user_id":1,"chat_id":2,"text":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer internal-token")
	rec := httptest.NewRecorder()

	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d with body %q", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func testConfig(debugEndpointsEnabled bool) config.Config {
	return config.Config{
		HTTPAddr:              ":0",
		RequestTimeout:        time.Second,
		WebhookSecret:         "webhook-secret",
		WebhookSecretHeader:   "X-Max-Webhook-Secret",
		InternalAPIToken:      "internal-token",
		DebugEndpointsEnabled: debugEndpointsEnabled,
	}
}
