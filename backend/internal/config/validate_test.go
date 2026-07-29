package config

import (
	"strings"
	"testing"
)

const strongSecret1 = "9f3a7c1e5b2d8f460a1c3e5f7b9d1c3e5f7b9d1c3e5f7b9d1c3e5f7b9d1c3e5f"
const strongSecret2 = "1c3e5f7b9d1c3e5f7b9d1c3e5f7b9d1c3e5f7b9d1c3e5f7b9d1c3e5f9f3a7c1e"

func TestValidateNoopOutsideProduction(t *testing.T) {
	cfg := &Config{Environment: "development", JWTSecret: "your-jwt-secret-here"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() outside production should be a no-op, got %v", err)
	}
}

func TestValidateRejectsMissingCredentialsEncryptionKeyInProduction(t *testing.T) {
	cfg := &Config{Environment: "production", JWTSecret: strongSecret1}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "CREDENTIALS_ENCRYPTION_KEY must be set explicitly") {
		t.Errorf("Validate() = %v, want an error about missing CREDENTIALS_ENCRYPTION_KEY", err)
	}
}

func TestValidateRejectsWeakJWTSecretInProduction(t *testing.T) {
	cfg := &Config{Environment: "production", JWTSecret: "short", CredentialsEncryptionKey: strongSecret2}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Errorf("Validate() = %v, want an error about a weak JWT_SECRET", err)
	}
}

func TestValidateRejectsPlaceholderSecretsInProduction(t *testing.T) {
	cases := []string{
		"change-me-to-a-random-64-char-secret-in-production",
		"your-jwt-secret-here-but-much-much-longer-than-32-chars",
		"REPLACE-ME-REPLACE-ME-REPLACE-ME-REPLACE-ME",
	}
	for _, secret := range cases {
		cfg := &Config{Environment: "production", JWTSecret: secret, CredentialsEncryptionKey: strongSecret2}
		if err := cfg.Validate(); err == nil {
			t.Errorf("Validate() with JWTSecret=%q should be rejected as a placeholder, got no error", secret)
		}
	}
}

func TestValidateRejectsIdenticalSecretsInProduction(t *testing.T) {
	cfg := &Config{Environment: "production", JWTSecret: strongSecret1, CredentialsEncryptionKey: strongSecret1}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "must not be the same value") {
		t.Errorf("Validate() = %v, want an error about identical secrets", err)
	}
}

func TestValidatePassesWithStrongDistinctSecretsInProduction(t *testing.T) {
	cfg := &Config{Environment: "production", JWTSecret: strongSecret1, CredentialsEncryptionKey: strongSecret2}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() with two strong, distinct secrets should pass, got %v", err)
	}
}
