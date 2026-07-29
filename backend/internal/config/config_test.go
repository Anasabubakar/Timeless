package config

import (
	"reflect"
	"testing"
)

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

func TestCORSOriginsMergesFrontendAndAllowedOrigins(t *testing.T) {
	cfg := &Config{
		FrontendURL:    "https://app.timeless.example",
		AllowedOrigins: []string{"https://staging.timeless.example", "https://app.timeless.example"},
	}
	got := cfg.CORSOrigins()
	want := []string{"https://app.timeless.example", "https://staging.timeless.example"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CORSOrigins() = %v, want %v (FrontendURL first, AllowedOrigins deduplicated against it)", got, want)
	}
}

func TestCORSOriginsHandlesEmptyAllowedOrigins(t *testing.T) {
	cfg := &Config{FrontendURL: "https://app.timeless.example"}
	got := cfg.CORSOrigins()
	want := []string{"https://app.timeless.example"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CORSOrigins() = %v, want %v", got, want)
	}
}

func TestCORSOriginsIgnoresBlankEntries(t *testing.T) {
	cfg := &Config{AllowedOrigins: []string{"", "https://app.timeless.example", ""}}
	got := cfg.CORSOrigins()
	want := []string{"https://app.timeless.example"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CORSOrigins() = %v, want %v", got, want)
	}
}
