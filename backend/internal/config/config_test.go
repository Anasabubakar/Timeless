package config

import "testing"

func TestCredentialKeyPrefersDedicatedSecret(t *testing.T) {
	cfg := &Config{JWTSecret: "jwt-secret", CredentialsEncryptionKey: "dedicated-secret"}
	if got := cfg.CredentialKey(); got != "dedicated-secret" {
		t.Errorf("CredentialKey() = %q, want %q", got, "dedicated-secret")
	}
}

func TestCredentialKeyFallsBackToJWTSecret(t *testing.T) {
	cfg := &Config{JWTSecret: "jwt-secret"}
	if got := cfg.CredentialKey(); got != "jwt-secret" {
		t.Errorf("CredentialKey() = %q, want %q", got, "jwt-secret")
	}
}

func TestIsDevelopmentIsProduction(t *testing.T) {
	dev := &Config{Environment: "development"}
	if !dev.IsDevelopment() || dev.IsProduction() {
		t.Errorf("expected development config to report IsDevelopment=true, IsProduction=false")
	}

	prod := &Config{Environment: "production"}
	if prod.IsDevelopment() || !prod.IsProduction() {
		t.Errorf("expected production config to report IsDevelopment=false, IsProduction=true")
	}
}
