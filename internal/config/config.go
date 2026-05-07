package config

import (
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
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:            env("HTTP_ADDR", ":8080"),
		WebhookSecretHeader: env("WEBHOOK_SECRET_HEADER", "X-Max-Webhook-Secret"),
		MAXBaseURL:          env("MAX_BASE_URL", "https://platform-api.max.ru"),
		OneCBaseURL:         os.Getenv("ONEC_BASE_URL"),
		WebhookSecret:       os.Getenv("WEBHOOK_SECRET"),
		InternalAPIToken:    os.Getenv("INTERNAL_API_TOKEN"),
		MAXToken:            os.Getenv("MAX_TOKEN"),
		OneCToken:           os.Getenv("ONEC_TOKEN"),
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
	if cfg.WebhookSecret == "" {
		return Config{}, fmt.Errorf("WEBHOOK_SECRET is required")
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
