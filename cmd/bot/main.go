package main

import (
	"errors"
	"log/slog"
	"net/http"
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
	onecAPI := onec.New(cfg.OneCBaseURL, cfg.OneCToken, cfg.RequestTimeout)
	botService := service.New(log, botStore, maxAPI, onecAPI)
	server := httpserver.New(cfg, log, botService)

	if err := server.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// maskDatabaseURL скрывает пароль в логах
func maskDatabaseURL(url string) string {
	if len(url) < 20 {
		return "***"
	}
	return url[:strings.Index(url, "://")+3] + "***@" + url[strings.LastIndex(url, "@")+1:]
}
