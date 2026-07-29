package logging

import (
	"strings"
	"testing"
)

func TestRedactAuthorizationHeader(t *testing.T) {
	in := `request failed: Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjMifQ.dGVzdHNpZ25hdHVyZQ was rejected`
	out := Redact(in)
	if strings.Contains(out, "eyJ") {
		t.Errorf("Redact(%q) = %q, still contains a raw JWT/bearer token", in, out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("Redact(%q) = %q, expected a [REDACTED] placeholder", in, out)
	}
}

func TestRedactPasswordKeyValue(t *testing.T) {
	in := `login failed for payload: {"email":"a@b.com","password":"hunter2annoying"}`
	out := Redact(in)
	if strings.Contains(out, "hunter2annoying") {
		t.Errorf("Redact(%q) = %q, still contains the raw password", in, out)
	}
	if !strings.Contains(out, "password=[REDACTED]") {
		t.Errorf("Redact(%q) = %q, expected the key name to be preserved as password=[REDACTED]", in, out)
	}
	if !strings.Contains(out, "a@b.com") {
		t.Errorf("Redact(%q) = %q, should not touch unrelated fields like email", in, out)
	}
}

func TestRedactAPIKey(t *testing.T) {
	in := "provider error: api_key=sk-abcdef1234567890 rate limited"
	out := Redact(in)
	if strings.Contains(out, "sk-abcdef1234567890") {
		t.Errorf("Redact(%q) = %q, still contains the raw API key", in, out)
	}
}

func TestRedactClientSecret(t *testing.T) {
	in := `oauth exchange failed: client_secret="a1b2c3d4e5f6g7h8" invalid_grant`
	out := Redact(in)
	if strings.Contains(out, "a1b2c3d4e5f6g7h8") {
		t.Errorf("Redact(%q) = %q, still contains the raw client secret", in, out)
	}
}

func TestRedactLeavesNonSensitiveTextAlone(t *testing.T) {
	in := "company created: id=abc123 name=Acme status=active"
	out := Redact(in)
	if out != in {
		t.Errorf("Redact(%q) = %q, expected no change for non-sensitive text", in, out)
	}
}
