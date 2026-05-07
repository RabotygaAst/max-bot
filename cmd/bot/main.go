package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

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

	memoryStore := store.NewMemoryStore()
	maxAPI := maxclient.New(cfg.MAXBaseURL, cfg.MAXToken, cfg.RequestTimeout)
	onecAPI := onec.New(cfg.OneCBaseURL, cfg.OneCToken, cfg.RequestTimeout)
	botService := service.New(log, memoryStore, maxAPI, onecAPI)
	server := httpserver.New(cfg, log, botService)

	if err := server.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
