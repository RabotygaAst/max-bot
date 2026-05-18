package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	maxclient "example.com/max-bot-go/internal/clients/max"
	"example.com/max-bot-go/internal/clients/onec"
	"example.com/max-bot-go/internal/config"
	"example.com/max-bot-go/internal/httpserver"
	"example.com/max-bot-go/internal/secret"
	"example.com/max-bot-go/internal/service"
	"example.com/max-bot-go/internal/store"
)

func main() {
	if isGenerateWebhookSecretMode(os.Args[1:]) {
		webhookSecret, err := secret.GenerateWebhookSecret(secret.DefaultWebhookSecretBytes)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(webhookSecret)
		return
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config error", "err", err)
		os.Exit(1)
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
		log.Info("using in-memory store (for development only)")
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

func isGenerateWebhookSecretMode(args []string) bool {
	return len(args) == 1 && (args[0] == "generate-webhook-secret" || args[0] == "--generate-webhook-secret")
}

// maskDatabaseURL скрывает пароль в логах
func maskDatabaseURL(url string) string {
	if len(url) < 20 {
		return "***"
	}
	return url[:strings.Index(url, "://")+3] + "***@" + url[strings.LastIndex(url, "@")+1:]
}
