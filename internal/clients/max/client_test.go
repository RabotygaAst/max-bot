package max

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendMessageWithKeyboard(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/messages" {
			t.Fatalf("path = %s, want /messages", r.URL.Path)
		}
		if r.URL.Query().Get("chat_id") != "42" {
			t.Fatalf("chat_id = %s, want 42", r.URL.Query().Get("chat_id"))
		}
		if r.Header.Get("Authorization") != "token" {
			t.Fatalf("authorization header = %q, want token", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL, "token", time.Second)
	err := client.SendMessageWithKeyboard(context.Background(), 42, "hello", Keyboard{{NewCallbackButton("Menu", "menu")}})
	if err != nil {
		t.Fatalf("SendMessageWithKeyboard returned error: %v", err)
	}

	if got["text"] != "hello" {
		t.Fatalf("text = %v, want hello", got["text"])
	}
	attachments, ok := got["attachments"].([]any)
	if !ok || len(attachments) != 1 {
		t.Fatalf("attachments = %#v, want one attachment", got["attachments"])
	}
	attachment, ok := attachments[0].(map[string]any)
	if !ok {
		t.Fatalf("attachment = %#v, want object", attachments[0])
	}
	if attachment["type"] != "inline_keyboard" {
		t.Fatalf("attachment type = %v, want inline_keyboard", attachment["type"])
	}
}
