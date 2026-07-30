package integration

import (
	"strings"
	"testing"
)

func TestOAuthProviderConfigured(t *testing.T) {
	configured := &OAuthProvider{ClientID: "id", ClientSecret: "secret"}
	if !configured.Configured() {
		t.Errorf("expected a provider with both client id and secret to be Configured()")
	}

	missingSecret := &OAuthProvider{ClientID: "id"}
	if missingSecret.Configured() {
		t.Errorf("expected a provider with no client secret to not be Configured()")
	}
}

func TestAuthorizeRedirectIncludesRequiredParams(t *testing.T) {
	p := &OAuthProvider{
		Provider:     "notion",
		ClientID:     "abc123",
		AuthorizeURL: "https://api.notion.com/v1/oauth/authorize",
	}
	redirectURI := p.AuthorizeRedirect("https://app.example.com/callback", "state-xyz", "")

	for _, want := range []string{
		"client_id=abc123",
		"response_type=code",
		"state=state-xyz",
		"owner=user", // Notion-specific: required for public integrations
	} {
		if !strings.Contains(redirectURI, want) {
			t.Errorf("AuthorizeRedirect() = %q, expected it to contain %q", redirectURI, want)
		}
	}
}

func TestAuthorizeRedirectOmitsOwnerForNonNotion(t *testing.T) {
	p := &OAuthProvider{Provider: "apollo", ClientID: "abc", AuthorizeURL: "https://app.apollo.io/#/oauth/authorize"}
	redirectURI := p.AuthorizeRedirect("https://app.example.com/callback", "state", "")
	if strings.Contains(redirectURI, "owner=user") {
		t.Errorf("expected owner=user to be Notion-specific, got %q", redirectURI)
	}
}

func TestAuthorizeRedirectOmitsPKCEParamsWhenDisabled(t *testing.T) {
	p := &OAuthProvider{Provider: "notion", ClientID: "abc", AuthorizeURL: "https://api.notion.com/v1/oauth/authorize"}
	redirectURI := p.AuthorizeRedirect("https://app.example.com/callback", "state", "some-verifier")
	if strings.Contains(redirectURI, "code_challenge") {
		t.Errorf("expected no code_challenge when PKCE is disabled, got %q", redirectURI)
	}
}

func TestAuthorizeRedirectIncludesPKCEParamsWhenEnabled(t *testing.T) {
	p := &OAuthProvider{Provider: "example", ClientID: "abc", AuthorizeURL: "https://example.com/authorize", PKCE: true}
	redirectURI := p.AuthorizeRedirect("https://app.example.com/callback", "state", "some-verifier")

	if !strings.Contains(redirectURI, "code_challenge_method=S256") {
		t.Errorf("expected code_challenge_method=S256, got %q", redirectURI)
	}
	// The challenge must be a deterministic function of the verifier, and
	// the raw verifier itself must never appear in the URL — it's only
	// ever sent server-to-server, at the token exchange.
	wantChallenge := pkceChallengeS256("some-verifier")
	if !strings.Contains(redirectURI, "code_challenge="+wantChallenge) {
		t.Errorf("expected code_challenge=%s, got %q", wantChallenge, redirectURI)
	}
	if strings.Contains(redirectURI, "some-verifier") {
		t.Errorf("the raw code_verifier must never appear in the authorize URL, got %q", redirectURI)
	}
}

func TestAuthorizeRedirectOmitsPKCEWhenVerifierEmpty(t *testing.T) {
	p := &OAuthProvider{Provider: "example", ClientID: "abc", AuthorizeURL: "https://example.com/authorize", PKCE: true}
	redirectURI := p.AuthorizeRedirect("https://app.example.com/callback", "state", "")
	if strings.Contains(redirectURI, "code_challenge") {
		t.Errorf("expected no code_challenge when no verifier was supplied, got %q", redirectURI)
	}
}

func TestGeneratePKCEVerifierIsUnique(t *testing.T) {
	a, err := GeneratePKCEVerifier()
	if err != nil {
		t.Fatalf("GeneratePKCEVerifier() error: %v", err)
	}
	b, err := GeneratePKCEVerifier()
	if err != nil {
		t.Fatalf("GeneratePKCEVerifier() error: %v", err)
	}
	if a == b {
		t.Error("expected two calls to GeneratePKCEVerifier to produce different values")
	}
	// RFC 7636 requires 43-128 characters.
	if len(a) < 43 || len(a) > 128 {
		t.Errorf("verifier length = %d, want 43-128 per RFC 7636", len(a))
	}
}

func TestPKCEChallengeS256IsDeterministic(t *testing.T) {
	verifier := "fixed-test-verifier-value"
	a := pkceChallengeS256(verifier)
	b := pkceChallengeS256(verifier)
	if a != b {
		t.Error("expected pkceChallengeS256 to be deterministic for the same verifier")
	}
	if a == verifier {
		t.Error("the challenge must not equal the raw verifier")
	}
}
