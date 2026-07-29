package config

import (
	"fmt"
	"strings"
)

// minSecretLength is the shortest a signing/encryption secret should be
// in production — 32 bytes/chars gives an HMAC-SHA256 key full entropy
// coverage; anything shorter is trivially weaker than the algorithm
// itself supports.
const minSecretLength = 32

// placeholderSecretMarkers catches the literal example values shipped in
// .env.example (there are two, with different wording, across the repo)
// and other common "someone forgot to change this" placeholders. A
// substring match, not exact — "change-me-to-a-random-64-char-secret-
// in-production" is 52 characters, long enough to pass a naive length
// check, but it's still the example text, not a real secret.
var placeholderSecretMarkers = []string{
	"change-me", "changeme", "change_me",
	"your-jwt-secret", "your-secret", "secret-here",
	"please-change", "replace-me", "example-secret",
	"password", "secret123", "test-secret",
}

// Validate rejects configuration that would be a genuine production
// security problem: a missing dedicated credential-encryption key
// (silently falling back to JWTSecret means one leaked secret
// compromises both token signing and credential-at-rest encryption), a
// weak or placeholder JWT/encryption secret, or the two secrets being
// identical. Deliberately a no-op outside production — a developer
// running against .env.example's placeholder JWT_SECRET locally is
// normal and shouldn't block starting the app.
func (c *Config) Validate() error {
	if !c.IsProduction() {
		return nil
	}

	var problems []string

	if err := checkSecretStrength("JWT_SECRET", c.JWTSecret); err != nil {
		problems = append(problems, err.Error())
	}

	if c.CredentialsEncryptionKey == "" {
		problems = append(problems, "CREDENTIALS_ENCRYPTION_KEY must be set explicitly in production — "+
			"leaving it unset means it silently falls back to JWT_SECRET, so a JWT_SECRET leak also "+
			"decrypts every stored OAuth token and integration credential")
	} else if err := checkSecretStrength("CREDENTIALS_ENCRYPTION_KEY", c.CredentialsEncryptionKey); err != nil {
		problems = append(problems, err.Error())
	}

	if c.CredentialsEncryptionKey != "" && c.CredentialsEncryptionKey == c.JWTSecret {
		problems = append(problems, "CREDENTIALS_ENCRYPTION_KEY must not be the same value as JWT_SECRET "+
			"in production — they protect different things and a leak of one shouldn't compromise the other")
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid configuration:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

func checkSecretStrength(name, value string) error {
	if value == "" {
		return nil // caught separately by env:"...,required" for JWT_SECRET
	}
	lower := strings.ToLower(value)
	for _, marker := range placeholderSecretMarkers {
		if strings.Contains(lower, marker) {
			return fmt.Errorf("%s looks like a placeholder/example value, not a real secret", name)
		}
	}
	if len(value) < minSecretLength {
		return fmt.Errorf("%s is only %d characters — use at least %d random characters (e.g. `openssl rand -hex 32`)", name, len(value), minSecretLength)
	}
	return nil
}
