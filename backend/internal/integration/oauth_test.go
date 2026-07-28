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
	redirectURI := p.AuthorizeRedirect("https://app.example.com/callback", "state-xyz")

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
	redirectURI := p.AuthorizeRedirect("https://app.example.com/callback", "state")
	if strings.Contains(redirectURI, "owner=user") {
		t.Errorf("expected owner=user to be Notion-specific, got %q", redirectURI)
	}
}
