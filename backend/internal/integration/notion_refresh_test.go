package integration

import (
	"context"
	"testing"
)

func TestNotionRefreshRequiresStoredRefreshToken(t *testing.T) {
	c := NewNotionClient("client-id", "client-secret")
	_, err := c.Refresh(context.Background(), map[string]string{"token": "access-token"})
	if err == nil {
		t.Fatalf("expected an error when no refresh_token is stored")
	}
}

func TestNotionRefreshRequiresOAuthClientConfigured(t *testing.T) {
	c := NewNotionClient("", "") // no client id/secret configured
	_, err := c.Refresh(context.Background(), map[string]string{"refresh_token": "rt"})
	if err == nil {
		t.Fatalf("expected an error when the OAuth client isn't configured")
	}
}
