package max

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientSendMessageSetsAuthorizationTokenWithoutBearer(t *testing.T) {
	const token = "test-max-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != token {
			t.Fatalf("Authorization header = %q, want %q", got, token)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type header = %q, want %q", got, "application/json")
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/messages" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/messages")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL, token, time.Second)
	if err := client.SendMessage(context.Background(), 123, "hello"); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
}
