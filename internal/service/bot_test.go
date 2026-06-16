package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	maxclient "example.com/max-bot-go/internal/clients/max"
	"example.com/max-bot-go/internal/clients/onec"
	"example.com/max-bot-go/internal/model"
	"example.com/max-bot-go/internal/store"
)

func TestAuthorizationFlowAcceptsPlainAccountNumber(t *testing.T) {
	var mu sync.Mutex
	var sentTexts []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/messages":
			var req struct {
				Text string `json:"text"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode message request: %v", err)
			}
			mu.Lock()
			sentTexts = append(sentTexts, req.Text)
			mu.Unlock()
			_, _ = w.Write([]byte(`{"success":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/max/v1/users/start":
			_, _ = w.Write([]byte(`{"success":true,"code":"OK","operation_id":"op-start","data":{}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/max/v1/accounts":
			_, _ = w.Write([]byte(`{"success":true,"code":"OK","operation_id":"op-accounts","data":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/max/v1/consents":
			_, _ = w.Write([]byte(`{"success":true,"code":"OK","operation_id":"op-consent","data":{}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/max/v1/account-link/start":
			_, _ = w.Write([]byte(`{"success":true,"code":"OK","operation_id":"op-link-start","data":{}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/max/v1/account-link/confirm":
			_, _ = w.Write([]byte(`{"success":true,"code":"OK","operation_id":"op-link-confirm","data":{"account_id":"ACC-000123456","number":"000123456","is_active":true}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	svc := New(
		slog.New(slog.NewTextHandler(testWriter{t: t}, nil)),
		store.NewMemoryStore(),
		maxclient.New(server.URL, "TEST_MAX_TOKEN", 5*time.Second),
		onec.New(server.URL, "TEST_ONEC_TOKEN", 5*time.Second),
	)

	ctx := context.Background()
	svc.ProcessUpdate(ctx, testUpdate("m-1", "/start"))
	svc.ProcessUpdate(ctx, testUpdate("m-2", actionAuthorize))
	svc.ProcessUpdate(ctx, testUpdate("m-3", "000123456"))
	svc.ProcessUpdate(ctx, testUpdate("m-4", "1234"))

	mu.Lock()
	defer mu.Unlock()
	if len(sentTexts) != 4 {
		t.Fatalf("expected 4 outgoing messages, got %d: %#v", len(sentTexts), sentTexts)
	}
	if !strings.Contains(sentTexts[0], "Авторизоваться") {
		t.Fatalf("start message should show authorization CTA, got: %q", sentTexts[0])
	}
	if strings.Contains(sentTexts[2], "не распознал") {
		t.Fatalf("plain account number was treated as unknown command: %q", sentTexts[2])
	}
	if !strings.Contains(sentTexts[2], "Лицевой счет найден") {
		t.Fatalf("account number should start SMS/code step, got: %q", sentTexts[2])
	}
	if !strings.Contains(sentTexts[3], "Лицевой счет привязан") {
		t.Fatalf("code should finish authorization, got: %q", sentTexts[3])
	}
}

func TestPlainAccountNumberStartsAuthorizationWithoutSession(t *testing.T) {
	var sentText string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/messages":
			var req struct {
				Text string `json:"text"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode message request: %v", err)
			}
			sentText = req.Text
			_, _ = w.Write([]byte(`{"success":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/max/v1/account-link/start":
			_, _ = w.Write([]byte(`{"success":true,"code":"OK","operation_id":"op-link-start","data":{}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	svc := New(
		slog.New(slog.NewTextHandler(testWriter{t: t}, nil)),
		store.NewMemoryStore(),
		maxclient.New(server.URL, "TEST_MAX_TOKEN", 5*time.Second),
		onec.New(server.URL, "TEST_ONEC_TOKEN", 5*time.Second),
	)

	svc.ProcessUpdate(context.Background(), testUpdate("m-account", "12345"))

	if strings.Contains(sentText, "не распознал") {
		t.Fatalf("plain account number without session was treated as unknown command: %q", sentText)
	}
	if !strings.Contains(sentText, "Лицевой счет найден") {
		t.Fatalf("plain account number should start authorization, got: %q", sentText)
	}
}

func TestAuthorizedUserMenuDoesNotRequestAuthorizationAgain(t *testing.T) {
	var mu sync.Mutex
	var sentTexts []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/messages":
			var req struct {
				Text string `json:"text"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode message request: %v", err)
			}
			mu.Lock()
			sentTexts = append(sentTexts, req.Text)
			mu.Unlock()
			_, _ = w.Write([]byte(`{"success":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/max/v1/users/start":
			_, _ = w.Write([]byte(`{"success":true,"code":"OK","operation_id":"op-start","data":{}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/max/v1/accounts":
			_, _ = w.Write([]byte(`{"success":true,"code":"OK","operation_id":"op-accounts","data":[{"account_id":"ACC-42","number":"00042","is_active":true}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	svc := New(
		slog.New(slog.NewTextHandler(testWriter{t: t}, nil)),
		store.NewMemoryStore(),
		maxclient.New(server.URL, "TEST_MAX_TOKEN", 5*time.Second),
		onec.New(server.URL, "TEST_ONEC_TOKEN", 5*time.Second),
	)

	ctx := context.Background()
	svc.ProcessUpdate(ctx, testUpdate("m-start-authorized", "/start"))
	svc.ProcessUpdate(ctx, testUpdate("m-menu-authorized", "меню"))

	mu.Lock()
	defer mu.Unlock()
	if len(sentTexts) != 2 {
		t.Fatalf("expected 2 outgoing messages, got %d: %#v", len(sentTexts), sentTexts)
	}
	for _, text := range sentTexts {
		if strings.Contains(text, "сначала нужно авторизоваться") || strings.Contains(text, "Отправьте номер лицевого счета") {
			t.Fatalf("authorized user should not be asked to authorize again, got: %q", text)
		}
	}
}

func TestAppealTextAfterCallbackButtonCreatesAppeal(t *testing.T) {
	var mu sync.Mutex
	var sentTexts []string
	var appealBody struct {
		Text string `json:"text"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/messages":
			var req struct {
				Text string `json:"text"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode message request: %v", err)
			}
			mu.Lock()
			sentTexts = append(sentTexts, req.Text)
			mu.Unlock()
			_, _ = w.Write([]byte(`{"success":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/max/v1/accounts":
			_, _ = w.Write([]byte(`{"success":true,"code":"OK","operation_id":"op-accounts","data":[{"account_id":"ACC-42","number":"00042","is_active":true}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/max/v1/accounts/ACC-42/appeals":
			if err := json.NewDecoder(r.Body).Decode(&appealBody); err != nil {
				t.Fatalf("decode appeal request: %v", err)
			}
			_, _ = w.Write([]byte(`{"success":true,"code":"OK","operation_id":"op-appeal","data":{"appeal_id":"APL-1","number":"ОБР-1","status":"зарегистрировано","sla":"3 рабочих дня"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	svc := New(
		slog.New(slog.NewTextHandler(testWriter{t: t}, nil)),
		store.NewMemoryStore(),
		maxclient.New(server.URL, "TEST_MAX_TOKEN", 5*time.Second),
		onec.New(server.URL, "TEST_ONEC_TOKEN", 5*time.Second),
	)

	ctx := context.Background()
	svc.ProcessUpdate(ctx, testCallbackUpdate("cb-appeal", actionAppealStart))
	svc.ProcessUpdate(ctx, testUpdate("m-appeal-text", "скацпфпрскрмкрф"))

	mu.Lock()
	defer mu.Unlock()
	if len(sentTexts) != 2 {
		t.Fatalf("expected 2 outgoing messages, got %d: %#v", len(sentTexts), sentTexts)
	}
	if strings.Contains(sentTexts[1], "не распознал") {
		t.Fatalf("appeal text after callback should not be treated as unknown command: %q", sentTexts[1])
	}
	if !strings.Contains(sentTexts[1], "Обращение зарегистрировано") {
		t.Fatalf("appeal text should create an appeal, got: %q", sentTexts[1])
	}
	if appealBody.Text != "скацпфпрскрмкрф" {
		t.Fatalf("unexpected appeal text sent to 1C: %q", appealBody.Text)
	}
}

func testUpdate(mid, text string) model.MAXUpdate {
	return model.MAXUpdate{
		UpdateType: "message_created",
		Message: model.MAXMessage{
			Sender:    model.MAXSender{UserID: 123456789, FirstName: "Иван"},
			Recipient: model.MAXRecipient{ChatID: 987654321},
			Body:      model.MAXBody{MID: mid, Text: text},
		},
	}
}

func testCallbackUpdate(callbackID, payload string) model.MAXUpdate {
	return model.MAXUpdate{
		UpdateType: "message_callback",
		Timestamp:  time.Now().Unix(),
		Callback: &model.Callback{
			CallbackID: callbackID,
			Payload:    payload,
			User:       &model.MAXSender{UserID: 123456789, FirstName: "Иван"},
			Message: &model.MAXMessage{
				Recipient: model.MAXRecipient{ChatID: 987654321},
			},
		},
	}
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimSpace(string(p)))
	return len(p), nil
}
