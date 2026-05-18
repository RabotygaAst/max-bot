package secret

import (
	"encoding/base64"
	"testing"
)

func TestGenerateWebhookSecretNotEmpty(t *testing.T) {
	secret, err := GenerateWebhookSecret(DefaultWebhookSecretBytes)
	if err != nil {
		t.Fatalf("GenerateWebhookSecret returned error: %v", err)
	}
	if secret == "" {
		t.Fatal("GenerateWebhookSecret returned an empty secret")
	}
}

func TestGenerateWebhookSecretReturnsDifferentValues(t *testing.T) {
	first, err := GenerateWebhookSecret(DefaultWebhookSecretBytes)
	if err != nil {
		t.Fatalf("first GenerateWebhookSecret returned error: %v", err)
	}
	second, err := GenerateWebhookSecret(DefaultWebhookSecretBytes)
	if err != nil {
		t.Fatalf("second GenerateWebhookSecret returned error: %v", err)
	}
	if first == second {
		t.Fatal("two consecutive generated secrets are equal")
	}
}

func TestGenerateWebhookSecretUsesRawURLEncoding(t *testing.T) {
	secret, err := GenerateWebhookSecret(DefaultWebhookSecretBytes)
	if err != nil {
		t.Fatalf("GenerateWebhookSecret returned error: %v", err)
	}
	if _, err := base64.RawURLEncoding.DecodeString(secret); err != nil {
		t.Fatalf("secret is not valid base64.RawURLEncoding: %v", err)
	}
}

func TestGenerateWebhookSecretDefaultEntropy(t *testing.T) {
	secret, err := GenerateWebhookSecret(DefaultWebhookSecretBytes)
	if err != nil {
		t.Fatalf("GenerateWebhookSecret returned error: %v", err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil {
		t.Fatalf("secret is not valid base64.RawURLEncoding: %v", err)
	}
	if len(decoded) != DefaultWebhookSecretBytes {
		t.Fatalf("decoded secret length = %d, want %d", len(decoded), DefaultWebhookSecretBytes)
	}
}

func TestGenerateWebhookSecretRejectsNonPositiveLength(t *testing.T) {
	if _, err := GenerateWebhookSecret(0); err == nil {
		t.Fatal("GenerateWebhookSecret(0) returned nil error")
	}
}
