package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	PollingLimit        int
	PollingTimeout      time.Duration
	PollingRetryDelay   time.Duration
	PollingTypes        []string
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
		WebhookSecretHeader: env("WEBHOOK_SECRET_HEADER", "X-Max-Bot-Api-Secret"),
		MAXBaseURL:          env("MAX_BASE_URL", "https://platform-api.max.ru"),
		OneCBaseURL:         os.Getenv("ONEC_BASE_URL"),
		WebhookSecret:       webhookSecret,
		InternalAPIToken:    os.Getenv("INTERNAL_API_TOKEN"),
		MAXToken:            os.Getenv("MAX_TOKEN"),
		OneCToken:           os.Getenv("ONEC_TOKEN"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
	}

	timeoutSeconds, err := strconv.Atoi(env("REQUEST_TIMEOUT_SECONDS", "10"))
	if err != nil || timeoutSeconds <= 0 {
		return Config{}, fmt.Errorf("REQUEST_TIMEOUT_SECONDS must be positive integer")
	}
	cfg.RequestTimeout = time.Duration(timeoutSeconds) * time.Second

	pollingLimit, err := strconv.Atoi(env("POLLING_LIMIT", "100"))
	if err != nil || pollingLimit <= 0 || pollingLimit > 1000 {
		return Config{}, fmt.Errorf("POLLING_LIMIT must be integer between 1 and 1000")
	}
	cfg.PollingLimit = pollingLimit

	pollingTimeoutSeconds, err := strconv.Atoi(env("POLLING_TIMEOUT_SECONDS", "30"))
	if err != nil || pollingTimeoutSeconds < 0 || pollingTimeoutSeconds > 90 {
		return Config{}, fmt.Errorf("POLLING_TIMEOUT_SECONDS must be integer between 0 and 90")
	}
	cfg.PollingTimeout = time.Duration(pollingTimeoutSeconds) * time.Second

	pollingRetrySeconds, err := strconv.Atoi(env("POLLING_RETRY_SECONDS", "5"))
	if err != nil || pollingRetrySeconds <= 0 {
		return Config{}, fmt.Errorf("POLLING_RETRY_SECONDS must be positive integer")
	}
	cfg.PollingRetryDelay = time.Duration(pollingRetrySeconds) * time.Second
	cfg.PollingTypes = splitCSV(env("POLLING_TYPES", "message_created,message_callback,bot_started"))

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

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func generateWebhookSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
