package main

import (
	"errors"
	"log/slog"
	"net/http"
	neturl "net/url"
	"os"
	"strings"

	maxclient "example.com/max-bot-go/internal/clients/max"
	"example.com/max-bot-go/internal/clients/onec"
	"example.com/max-bot-go/internal/config"
	"example.com/max-bot-go/internal/httpserver"
	"example.com/max-bot-go/internal/service"
	"example.com/max-bot-go/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config error", "err", err)
		os.Exit(1)
	}
	if os.Getenv("WEBHOOK_SECRET") == "" {
		log.Warn("WEBHOOK_SECRET was not set, generated new value for this launch", "webhook_secret", cfg.WebhookSecret)
	}

	// Инициализируем хранилище
	var botStore store.Store
	if cfg.DatabaseURL != "" {
		log.Info("using PostgreSQL store", "database_url", maskDatabaseURL(cfg.DatabaseURL))
		pgStore, err := store.NewPostgresStore(cfg.DatabaseURL)
		if err != nil {
			log.Error("postgres init error", "err", err)
			os.Exit(1)
		}
		defer pgStore.Close()
		botStore = pgStore
	} else {
		log.Warn("using in-memory store: state will be lost after restart")
		botStore = store.NewMemoryStore()
	}

	maxAPI := maxclient.New(cfg.MAXBaseURL, cfg.MAXToken, cfg.RequestTimeout)
	logOneCMode(log, cfg.OneCBaseURL)
	onecAPI := onec.New(cfg.OneCBaseURL, cfg.OneCToken, cfg.RequestTimeout)
	botService := service.New(log, botStore, maxAPI, onecAPI)
	server := httpserver.New(cfg, log, botService)

	if err := server.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// maskDatabaseURL скрывает пароль в логах
func maskDatabaseURL(rawURL string) string {
	parsedURL, err := neturl.Parse(rawURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" || parsedURL.User == nil {
		return "***"
	}

	schemeIdx := strings.Index(rawURL, "://")
	atIdx := strings.LastIndex(rawURL, "@")
	if schemeIdx == -1 || atIdx == -1 || schemeIdx+3 >= atIdx {
		return "***"
	}

	return rawURL[:schemeIdx+3] + "***@" + rawURL[atIdx+1:]
}

func logOneCMode(log *slog.Logger, baseURL string) {
	lower := strings.ToLower(baseURL)
	if strings.Contains(lower, "localhost:1080") || strings.Contains(lower, "mock-onec") {
		log.Warn("ONEC_BASE_URL points to mock 1C mode; mock-onec-config.json is test-only and is not a real data source", "onec_base_url", baseURL)
		return
	}
	log.Info("ONEC_BASE_URL points to real 1C HTTP service; business data must come from 1C", "onec_base_url", baseURL)
}
