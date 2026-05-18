package httpserver

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"example.com/max-bot-go/internal/config"
	"example.com/max-bot-go/internal/model"
	"example.com/max-bot-go/internal/service"
)

type Server struct {
	cfg     config.Config
	log     *slog.Logger
	service *service.BotService
	server  *http.Server
}

func New(cfg config.Config, log *slog.Logger, service *service.BotService) *Server {
	s := &Server{cfg: cfg, log: log, service: service}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/webhook/max", s.maxWebhook)
	mux.HandleFunc("/internal/notifications/send", s.sendNotification)
	if cfg.DebugEndpointsEnabled {
		mux.HandleFunc("/debug/send-test-update", s.debugSendTestUpdate)
	}
	s.server = &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           requestLog(log, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

func (s *Server) Run() error {
	s.log.Info("http server started", "addr", s.cfg.HTTPAddr)
	return s.server.ListenAndServe()
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) maxWebhook(w http.ResponseWriter, r *http.Request) {
	if !s.validWebhookSecret(r) {
		s.log.Warn("webhook rejected: invalid secret")
		writeJSON(w, http.StatusUnauthorized, map[string]any{"success": false})
		return
	}

	body := http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	var upd model.MAXUpdate
	if err := json.NewDecoder(body).Decode(&upd); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "invalid json"})
		return
	}

	// Webhook отвечает быстро, а сценарий обрабатывается отдельно.
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.RequestTimeout*3)
	go func() {
		defer cancel()
		s.service.ProcessUpdate(ctx, upd)
	}()
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) debugSendTestUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.validInternalToken(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"success": false})
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"success": false, "message": "method not allowed"})
		return
	}

	var req struct {
		UserID     int64  `json:"user_id"`
		ChatID     int64  `json:"chat_id"`
		Text       string `json:"text"`
		MID        string `json:"mid,omitempty"`
		UpdateType string `json:"update_type,omitempty"`
	}

	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "invalid json"})
		return
	}

	if req.UserID == 0 || req.ChatID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "user_id and chat_id are required"})
		return
	}

	if req.Text == "" {
		req.Text = "/start"
	}
	if req.UpdateType == "" {
		req.UpdateType = "message_created"
	}
	if req.MID == "" {
		req.MID = fmt.Sprintf("debug-%d", time.Now().UnixNano())
	}

	upd := model.MAXUpdate{
		UpdateType: req.UpdateType,
		Timestamp:  time.Now().UnixMilli(),
		Message: model.MAXMessage{
			Sender: model.MAXSender{
				UserID: req.UserID,
			},
			Recipient: model.MAXRecipient{
				ChatID: req.ChatID,
			},
			Body: model.MAXBody{
				MID:  req.MID,
				Text: req.Text,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.RequestTimeout*3)
	go func() {
		defer cancel()
		s.service.ProcessUpdate(ctx, upd)
	}()

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) sendNotification(w http.ResponseWriter, r *http.Request) {
	if !s.validInternalToken(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"success": false})
		return
	}

	var req model.NotificationRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "invalid json"})
		return
	}
	if err := s.service.SendNotification(r.Context(), req); err != nil {
		s.log.Error("send notification failed", "operation_id", req.OperationID, "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{"success": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) validWebhookSecret(r *http.Request) bool {
	got := r.Header.Get(s.cfg.WebhookSecretHeader)
	if got == "" {
		// Резервные имена заголовков для разных прокси/интеграций. Секрет в URL не поддерживается.
		got = r.Header.Get("X-Webhook-Secret")
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(s.cfg.WebhookSecret)) == 1
}

func (s *Server) validInternalToken(r *http.Request) bool {
	got := r.Header.Get("Authorization")
	want := "Bearer " + s.cfg.InternalAPIToken
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func requestLog(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Info("http request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(started).Milliseconds())
	})
}
