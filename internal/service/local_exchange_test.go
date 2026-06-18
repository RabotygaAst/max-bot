package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	maxclient "example.com/max-bot-go/internal/clients/max"
	"example.com/max-bot-go/internal/clients/onec"
	"example.com/max-bot-go/internal/model"
	"example.com/max-bot-go/internal/store"
)

const (
	exchangeUserID = int64(123456789)
	exchangeChatID = int64(987654321)
	exchangeToken  = "TEST_ONEC_TOKEN"
	mockAccountID  = "MOCK-ACC-000123456"
)

type recordedOneCRequest struct {
	Method    string
	Path      string
	RawQuery  string
	Header    http.Header
	Body      map[string]any
	Timestamp time.Time
}

type sentMAXMessage struct {
	Text    string `json:"text"`
	Buttons []struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Payload string `json:"payload"`
	}
}

type exchangeHarness struct {
	t      *testing.T
	server *httptest.Server
	svc    *BotService
	store  store.Store
	mu     sync.Mutex
	onec   []recordedOneCRequest
	max    []sentMAXMessage
}

func newExchangeHarness(t *testing.T) *exchangeHarness {
	t.Helper()
	h := &exchangeHarness{t: t, store: store.NewMemoryStore()}
	h.server = httptest.NewServer(http.HandlerFunc(h.handleHTTP))
	h.svc = New(slog.New(slog.NewTextHandler(testWriter{t: t}, nil)), h.store, maxclient.New(h.server.URL, "TEST_MAX_TOKEN", 5*time.Second), onec.New(h.server.URL, exchangeToken, 5*time.Second))
	t.Cleanup(h.server.Close)
	return h
}

func (h *exchangeHarness) handleHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost && r.URL.Path == "/messages" {
		var raw struct {
			Text        string `json:"text"`
			Attachments []struct {
				Payload struct {
					Buttons [][]struct{ Type, Text, Payload string } `json:"buttons"`
				} `json:"payload"`
			} `json:"attachments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			h.t.Fatalf("decode MAX message: %v", err)
		}
		msg := sentMAXMessage{Text: raw.Text}
		for _, a := range raw.Attachments {
			for _, row := range a.Payload.Buttons {
				for _, b := range row {
					msg.Buttons = append(msg.Buttons, struct {
						Type    string `json:"type"`
						Text    string `json:"text"`
						Payload string `json:"payload"`
					}{Type: b.Type, Text: b.Text, Payload: b.Payload})
				}
			}
		}
		h.mu.Lock()
		h.max = append(h.max, msg)
		h.mu.Unlock()
		_, _ = w.Write([]byte(`{"success":true}`))
		return
	}
	if got := r.Header.Get("Authorization"); got != "Bearer "+exchangeToken {
		http.Error(w, "bad auth", http.StatusUnauthorized)
		return
	}
	bodyBytes, _ := io.ReadAll(r.Body)
	body := map[string]any{}
	if len(strings.TrimSpace(string(bodyBytes))) > 0 {
		_ = json.Unmarshal(bodyBytes, &body)
	}
	h.mu.Lock()
	h.onec = append(h.onec, recordedOneCRequest{Method: r.Method, Path: r.URL.Path, RawQuery: r.URL.RawQuery, Header: r.Header.Clone(), Body: body, Timestamp: time.Now()})
	h.mu.Unlock()
	h.writeOneCResponse(w, r)
}

func (h *exchangeHarness) writeOneCResponse(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	var payload string
	switch {
	case r.Method == http.MethodPost && p == "/max/v1/users/start":
		payload = `{}`
	case r.Method == http.MethodGet && p == "/max/v1/accounts":
		payload = `[]`
	case r.Method == http.MethodPost && p == "/max/v1/consents":
		payload = `{}`
	case r.Method == http.MethodPost && p == "/max/v1/account-link/start":
		payload = `{}`
	case r.Method == http.MethodPost && p == "/max/v1/account-link/confirm":
		payload = `{"account_id":"MOCK-ACC-000123456","number":"000123456","address":"MOCK адрес","is_active":true}`
	case r.Method == http.MethodGet && p == "/max/v1/accounts/MOCK-ACC-000123456/balance":
		payload = `{"account_id":"MOCK-ACC-000123456","debt":123.45,"overpay":0,"currency":"руб.","actual_at":"2026-06-18"}`
	case r.Method == http.MethodGet && p == "/max/v1/accounts/MOCK-ACC-000123456/meters":
		payload = `[{"meter_id":"MOCK-HVS-001","display_name":"ХВС","resource":"water","unit":"м3","last_value":244.1,"last_reading_date":"2026-05-20","can_submit":true},{"meter_id":"MOCK-GVS-001","display_name":"ГВС","unit":"м3","last_value":100,"can_submit":true}]`
	case r.Method == http.MethodPost && p == "/max/v1/accounts/MOCK-ACC-000123456/meters/MOCK-HVS-001/readings":
		payload = `{"document_number":"MOCK-РП-000001","document_date":"2026-06-18","posted":false,"meter_id":"MOCK-HVS-001","value":245.678,"status":"created"}`
	case r.Method == http.MethodGet && p == "/max/v1/accounts/MOCK-ACC-000123456/invoice":
		payload = `{"account_id":"MOCK-ACC-000123456","period":"2026-06","amount":123.45,"currency":"руб.","document_number":"MOCK-КВ-000001","document_date":"2026-06-18","download_url":"https://mock.local/MOCK-КВ-000001.pdf"}`
	case r.Method == http.MethodPost && p == "/max/v1/accounts/MOCK-ACC-000123456/payment-link":
		payload = `{"account_id":"MOCK-ACC-000123456","amount":123.45,"currency":"руб.","payment_url":"https://mock.local/pay/MOCK-ACC-000123456","expires_at":"2026-06-19T00:00:00Z"}`
	case r.Method == http.MethodPost && p == "/max/v1/accounts/MOCK-ACC-000123456/appeals":
		payload = `{"appeal_id":"MOCK-APL-001","number":"MOCK-APL-001","status":"принято","sla":"3 дня"}`
	case r.Method == http.MethodGet && p == "/max/v1/accounts/MOCK-ACC-000123456/outages":
		payload = `[{"outage_id":"MOCK-OUT-001","resource":"ХВС","status":"planned","address":"MOCK адрес","starts_at":"2026-06-20T10:00:00Z","ends_at":"2026-06-20T12:00:00Z","reason":"MOCK работы","comment":"MOCK уведомление"}]`
	case r.Method == http.MethodGet && p == "/max/v1/reference/appointment-topics":
		payload = `[{"topic_id":"billing","title":"Начисления и оплата"}]`
	case r.Method == http.MethodPost && p == "/max/v1/accounts/MOCK-ACC-000123456/appointments":
		payload = `{"appointment_id":"MOCK-APT-001","number":"MOCK-APT-001","topic_title":"Начисления и оплата","office_address":"MOCK офис","starts_at":"2026-06-21T09:00:00Z","status":"created"}`
	case r.Method == http.MethodGet && p == "/max/v1/reference/organization":
		payload = `{"name":"MOCK Организация","phone":"+7 000 000-00-00","email":"mock@example.test","site":"https://mock.local","work_hours":"MOCK 9-18","office_address":"MOCK офис","customer_service_hours":"MOCK прием"}`
	case r.Method == http.MethodGet && p == "/max/v1/reference/emergency":
		payload = `{"dispatcher_phone":"MOCK-Диспетчер","emergency_phone":"MOCK-Аварийная","gas_phone":"MOCK-Газ","electricity_phone":"MOCK-Свет","comment":"MOCK звоните 112"}`
	case r.Method == http.MethodGet && p == "/max/v1/reference/help":
		payload = `{"text":"❓ Помощь MOCK: используйте кнопки"}`
	default:
		h.t.Fatalf("unexpected ONEC request: %s %s", r.Method, r.URL.String())
	}
	_, _ = fmt.Fprintf(w, `{"success":true,"code":"OK","operation_id":"op-%d","data":%s}`, time.Now().UnixNano(), payload)
}

func (h *exchangeHarness) text(mid, text string) {
	h.svc.ProcessUpdate(context.Background(), testUpdate(mid, text))
}
func (h *exchangeHarness) callback(id, payload string) {
	h.svc.ProcessUpdate(context.Background(), model.MAXUpdate{Timestamp: time.Now().Unix(), Callback: &model.Callback{CallbackID: id, Payload: payload, User: &model.MAXSender{UserID: exchangeUserID, FirstName: "MOCK"}, Message: &model.MAXMessage{Recipient: model.MAXRecipient{ChatID: exchangeChatID}}}})
}

func (h *exchangeHarness) request(method, path string) (recordedOneCRequest, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.onec {
		if r.Method == method && r.Path == path {
			return r, true
		}
	}
	return recordedOneCRequest{}, false
}

func (h *exchangeHarness) requireBodyField(r recordedOneCRequest, key string) {
	h.t.Helper()
	if r.Body[key] == nil || fmt.Sprint(r.Body[key]) == "" {
		h.t.Fatalf("%s %s missing body field %s: %#v", r.Method, r.Path, key, r.Body)
	}
}

func (h *exchangeHarness) requireTextContains(s string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, m := range h.max {
		if strings.Contains(m.Text, s) {
			return
		}
	}
	h.t.Fatalf("MAX text containing %q not sent; messages=%#v", s, h.max)
}

func runFullLocalExchangeScenario(t *testing.T) *exchangeHarness {
	h := newExchangeHarness(t)
	h.text("m-start", "/start")
	h.callback("cb-authorize", actionAuthorize)
	h.text("m-account", "000123456")
	h.text("m-code", "1234")
	h.callback("cb-balance", actionBalance)
	h.callback("cb-meters", actionMeters)
	h.text("m-reading", "показание MOCK-HVS-001 245.678")
	h.callback("cb-invoice", actionInvoice)
	h.text("m-invoice-period", "квитанция 2026-06")
	h.callback("cb-payment", actionPayment)
	h.callback("cb-appeal", actionAppealStart)
	h.text("m-appeal", "обращение тестовое обращение из локального обмена")
	h.callback("cb-outages", actionOutages)
	h.callback("cb-appointment", actionAppointment)
	h.text("m-appointment", "запись billing")
	h.callback("cb-organization", actionOrganization)
	h.callback("cb-emergency", actionEmergency)
	h.callback("cb-help", actionHelp)
	return h
}

func TestLocalExchangeFullUserScenario(t *testing.T) {
	h := runFullLocalExchangeScenario(t)
	for _, phrase := range []string{"Здравствуйте", "Авторизация", "Лицевой счет найден", "Лицевой счет привязан", "Баланс", "Показания", "Показание зарегистрировано", "Квитанция", "Оплата", "Обращение зарегистрировано", "Отключения", "Запись на прием", "Организация", "Аварийная", "Помощь"} {
		h.requireTextContains(phrase)
	}
}

func TestGuestButtonsCallbacks(t *testing.T) {
	h := newExchangeHarness(t)
	h.text("guest-start", "/start")
	want := map[string]string{"Авторизоваться": actionAuthorize, "организации": actionOrganization, "Аварийная": actionEmergency, "Помощь": actionHelp}
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.max) == 0 {
		t.Fatal("no MAX messages")
	}
	buttons := h.max[len(h.max)-1].Buttons
	for label, payload := range want {
		found := false
		for _, b := range buttons {
			if strings.Contains(b.Text, label) && b.Payload == payload && b.Payload != "" {
				found = true
			}
		}
		if !found {
			t.Fatalf("guest button %q payload %q not found in %#v", label, payload, buttons)
		}
	}
}

func TestAuthorizedButtonsCallbacks(t *testing.T) {
	h := newExchangeHarness(t)
	h.text("s1", "/start")
	h.callback("a1", actionAuthorize)
	h.text("a2", "000123456")
	h.text("a3", "1234")
	want := map[string]string{"Баланс": actionBalance, "Квитанция": actionInvoice, "Показания": actionMeters, "Оплатить": actionPayment, "Обращение": actionAppealStart, "Отключения": actionOutages, "Запись": actionAppointment, "Организация": actionOrganization, "Аварийная": actionEmergency, "Помощь": actionHelp}
	h.mu.Lock()
	buttons := h.max[len(h.max)-1].Buttons
	h.mu.Unlock()
	for label, payload := range want {
		found := false
		for _, b := range buttons {
			if strings.Contains(b.Text, label) && b.Payload == payload && b.Payload != "" {
				found = true
			}
		}
		if !found {
			t.Fatalf("authorized button %q payload %q not found in %#v", label, payload, buttons)
		}
	}
	for _, payload := range []string{actionBalance, actionMeters, actionPayment, actionOutages, actionAppointment} {
		h.callback("probe-"+payload, payload)
	}
}

func TestOneCRequestsReachedMock(t *testing.T) {
	h := runFullLocalExchangeScenario(t)
	for _, e := range []struct {
		m, p      string
		queryKeys []string
		bodyKeys  []string
	}{
		{"POST", "/max/v1/users/start", nil, []string{"max_user_id", "chat_id", "source"}},
		{"POST", "/max/v1/consents", nil, []string{"max_user_id", "consent_version", "source"}},
		{"POST", "/max/v1/account-link/start", nil, []string{"max_user_id", "account_number", "source"}},
		{"POST", "/max/v1/account-link/confirm", nil, []string{"max_user_id", "account_number", "code", "source"}},
		{"GET", "/max/v1/accounts/MOCK-ACC-000123456/balance", []string{"max_user_id"}, nil},
		{"GET", "/max/v1/accounts/MOCK-ACC-000123456/meters", []string{"max_user_id"}, nil},
		{"POST", "/max/v1/accounts/MOCK-ACC-000123456/meters/MOCK-HVS-001/readings", nil, []string{"max_user_id", "message_id", "operation_id", "source"}},
		{"GET", "/max/v1/accounts/MOCK-ACC-000123456/invoice", []string{"period", "max_user_id"}, nil},
		{"POST", "/max/v1/accounts/MOCK-ACC-000123456/payment-link", nil, []string{"max_user_id", "operation_id", "source"}},
		{"POST", "/max/v1/accounts/MOCK-ACC-000123456/appeals", nil, []string{"max_user_id", "message_id", "operation_id", "source"}},
		{"GET", "/max/v1/accounts/MOCK-ACC-000123456/outages", []string{"max_user_id"}, nil},
		{"GET", "/max/v1/reference/appointment-topics", nil, nil},
		{"POST", "/max/v1/accounts/MOCK-ACC-000123456/appointments", nil, []string{"max_user_id", "topic_id", "operation_id", "source"}},
		{"GET", "/max/v1/reference/organization", nil, nil},
		{"GET", "/max/v1/reference/emergency", nil, nil},
		{"GET", "/max/v1/reference/help", nil, nil},
	} {
		r, ok := h.request(e.m, e.p)
		if !ok {
			t.Fatalf("request did not reach ONEC mock: %s %s", e.m, e.p)
		}
		if r.Header.Get("Authorization") != "Bearer "+exchangeToken {
			t.Fatalf("bad auth for %s %s", e.m, e.p)
		}
		values, err := url.ParseQuery(r.RawQuery)
		if err != nil {
			t.Fatalf("bad raw query for %s %s: %v", e.m, e.p, err)
		}
		for _, key := range e.queryKeys {
			if values.Get(key) == "" {
				t.Fatalf("%s %s missing query %s in %q", e.m, e.p, key, r.RawQuery)
			}
		}
		for _, key := range e.bodyKeys {
			h.requireBodyField(r, key)
		}
	}
}

func TestReadingRequestContainsOperationID(t *testing.T) {
	h := runFullLocalExchangeScenario(t)
	r, ok := h.request("POST", "/max/v1/accounts/MOCK-ACC-000123456/meters/MOCK-HVS-001/readings")
	if !ok {
		t.Fatal("reading request missing")
	}
	for _, k := range []string{"operation_id", "message_id", "source", "max_user_id"} {
		if fmt.Sprint(r.Body[k]) == "" || r.Body[k] == nil {
			t.Fatalf("reading body missing %s: %#v", k, r.Body)
		}
	}
	if r.Body["source"] != "MAX" {
		t.Fatalf("bad source: %#v", r.Body)
	}
}

func TestAccountLinkIsReusedAfterStart(t *testing.T) {
	h := runFullLocalExchangeScenario(t)
	before := len(h.onec)
	h.text("reuse-start", "/start")
	h.callback("reuse-balance", actionBalance)
	after := len(h.onec)
	if after-before != 2 {
		t.Fatalf("expected only start and balance after reused link, got %d new requests", after-before)
	}
	if _, ok := h.request("GET", "/max/v1/accounts/MOCK-ACC-000123456/balance"); !ok {
		t.Fatal("balance request missing")
	}
}
