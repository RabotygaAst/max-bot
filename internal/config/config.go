package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr            string
	RequestTimeout      time.Duration
	WebhookSecret       string
	WebhookSecretHeader string
	InternalAPIToken    string
	MAXBaseURL          string
	MAXToken            string
	OneCBaseURL         string
	OneCToken           string
	DatabaseURL         string
}

func Load() (Config, error) {
	webhookSecret := os.Getenv("WEBHOOK_SECRET")
	if webhookSecret == "" {
		generated, err := generateWebhookSecret()
		if err != nil {
			return Config{}, fmt.Errorf("generate WEBHOOK_SECRET: %w", err)
		}
		webhookSecret = generated
	}

	cfg := Config{
		HTTPAddr:            env("HTTP_ADDR", ":8080"),
		WebhookSecretHeader: env("WEBHOOK_SECRET_HEADER", "X-Max-Bot-Api-Secret"), MAXBaseURL: env("MAX_BASE_URL", "https://platform-api.max.ru"),
		OneCBaseURL:      os.Getenv("ONEC_BASE_URL"),
		WebhookSecret:    webhookSecret,
		InternalAPIToken: os.Getenv("INTERNAL_API_TOKEN"),
		MAXToken:         os.Getenv("MAX_TOKEN"),
		OneCToken:        os.Getenv("ONEC_TOKEN"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
	}

	timeoutSeconds, err := strconv.Atoi(env("REQUEST_TIMEOUT_SECONDS", "10"))
	if err != nil || timeoutSeconds <= 0 {
		return Config{}, fmt.Errorf("REQUEST_TIMEOUT_SECONDS must be positive integer")
	}
	cfg.RequestTimeout = time.Duration(timeoutSeconds) * time.Second

	if cfg.MAXToken == "" {
		return Config{}, fmt.Errorf("MAX_TOKEN is required")
	}
	if cfg.OneCBaseURL == "" {
		return Config{}, fmt.Errorf("ONEC_BASE_URL is required")
	}
	if cfg.OneCToken == "" {
		return Config{}, fmt.Errorf("ONEC_TOKEN is required")
	}
	if cfg.InternalAPIToken == "" {
		return Config{}, fmt.Errorf("INTERNAL_API_TOKEN is required")
	}

	return cfg, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func generateWebhookSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
