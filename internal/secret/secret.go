package secret

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const DefaultWebhookSecretBytes = 32

// GenerateWebhookSecret returns a cryptographically-random, URL-safe secret.
func GenerateWebhookSecret(bytes int) (string, error) {
	if bytes <= 0 {
		return "", fmt.Errorf("bytes must be positive")
	}

	randomBytes := make([]byte, bytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}
