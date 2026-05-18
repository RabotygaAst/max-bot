package httpserver

import (
	"bytes"
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

const validWebhookPayload = `{
	"update_type":"message_created",
	"timestamp":1710000000000,
	"message":{
		"sender":{"user_id":123,"first_name":"Test"},
		"recipient":{"chat_id":456},
		"body":{"mid":"mid-1","text":"unknown command"}
	}
}`

func TestMaxWebhookRejectsGet(t *testing.T) {
	s := newTestServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/webhook/max", nil)
	req.Header.Set("X-Max-Webhook-Secret", "secret")
	rr := httptest.NewRecorder()

	s.maxWebhook(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
	if got := rr.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("Allow header = %q, want %q", got, http.MethodPost)
	}
}

func TestMaxWebhookAcceptsPostWithValidSecret(t *testing.T) {
	maxCalled := make(chan struct{}, 1)
	s := newTestServer(t, maxCalled)
	req := httptest.NewRequest(http.MethodPost, "/webhook/max", bytes.NewBufferString(validWebhookPayload))
	req.Header.Set("X-Max-Webhook-Secret", "secret")
	rr := httptest.NewRecorder()

	s.maxWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	select {
	case <-maxCalled:
	case <-time.After(time.Second):
		t.Fatal("expected webhook update to be processed")
	}
}

func TestMaxWebhookRejectsPostWithInvalidSecret(t *testing.T) {
	s := newTestServer(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/webhook/max", bytes.NewBufferString(validWebhookPayload))
	req.Header.Set("X-Max-Webhook-Secret", "wrong-secret")
	rr := httptest.NewRecorder()

	s.maxWebhook(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func newTestServer(t *testing.T, maxCalled chan<- struct{}) *Server {
	t.Helper()

	maxAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case maxCalled <- struct{}{}:
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(maxAPI.Close)

	onecAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	t.Cleanup(onecAPI.Close)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Config{
		RequestTimeout:      time.Second,
		WebhookSecret:       "secret",
		WebhookSecretHeader: "X-Max-Webhook-Secret",
	}
	botService := service.New(
		logger,
		store.NewMemoryStore(),
		maxclient.New(maxAPI.URL, "token", time.Second),
		onec.New(onecAPI.URL, "token", time.Second),
	)

	return &Server{cfg: cfg, log: logger, service: botService}
}
