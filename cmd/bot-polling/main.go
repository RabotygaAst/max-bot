package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	maxclient "example.com/max-bot-go/internal/clients/max"
	"example.com/max-bot-go/internal/clients/onec"
	"example.com/max-bot-go/internal/config"
	"example.com/max-bot-go/internal/polling"
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

	botStore, closeStore := initStore(log, cfg.DatabaseURL)
	defer closeStore()

	maxAPI := maxclient.New(cfg.MAXBaseURL, cfg.MAXToken, cfg.RequestTimeout+cfg.PollingTimeout)
	logOneCMode(log, cfg.OneCBaseURL)
	onecAPI := onec.New(cfg.OneCBaseURL, cfg.OneCToken, cfg.RequestTimeout)
	botService := service.New(log, botStore, maxAPI, onecAPI)
	poller := polling.New(log, maxAPI, botService, cfg.PollingLimit, cfg.PollingTimeout, cfg.PollingRetryDelay, cfg.PollingTypes)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := poller.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("polling stopped", "err", err)
		os.Exit(1)
	}
}

func initStore(log *slog.Logger, databaseURL string) (store.Store, func()) {
	if databaseURL == "" {
		log.Info("using in-memory store (for development only)")
		return store.NewMemoryStore(), func() {}
	}

	log.Info("using PostgreSQL store", "database_url", maskDatabaseURL(databaseURL))
	pgStore, err := store.NewPostgresStore(databaseURL)
	if err != nil {
		log.Error("postgres init error", "err", err)
		os.Exit(1)
	}
	return pgStore, func() { _ = pgStore.Close() }
}

func maskDatabaseURL(url string) string {
	if len(url) < 20 {
		return "***"
	}
	return url[:strings.Index(url, "://")+3] + "***@" + url[strings.LastIndex(url, "@")+1:]
}

func logOneCMode(log *slog.Logger, baseURL string) {
	lower := strings.ToLower(baseURL)
	if strings.Contains(lower, "localhost:1080") || strings.Contains(lower, "mock-onec") {
		log.Warn("ONEC_BASE_URL points to mock 1C mode; mock-onec-config.json is test-only and is not a real data source", "onec_base_url", baseURL)
		return
	}
	log.Info("ONEC_BASE_URL points to real 1C HTTP service; business data must come from 1C", "onec_base_url", baseURL)
}
