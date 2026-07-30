package handler

import (
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"

	"github.com/timeless/backend/internal/config"
)

func signToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return s
}

// redirectError runs a GET against app and returns the "error" query
// param of the redirect it produces (empty string if there wasn't one).
func redirectError(t *testing.T, app *fiber.App, target string) string {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("GET", target, nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Fatalf("status = %d, want %d (redirect)", resp.StatusCode, fiber.StatusFound)
	}
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("could not parse redirect Location %q: %v", resp.Header.Get("Location"), err)
	}
	return loc.Query().Get("error")
}

// TestOAuthStartRejectsInvalidTokens is the regression test for the JWT
// hardening fix: Start's keyfunc previously just returned
// []byte(cfg.JWTSecret), with no algorithm check and no keyring lookup.
// Every case here must redirect with an error and never reach Redis —
// h is constructed with a nil redis.Client, which would panic if the
// handler tried to use it, so "the test doesn't panic" is itself part
// of what's being verified.
func TestOAuthStartRejectsInvalidTokens(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-at-least-32-characters-long", FrontendURL: "https://app.example.com"}
	h := NewOAuthHandler(cfg, nil, nil, nil)
	h.providers["notion"].ClientID = "test-client-id"
	h.providers["notion"].ClientSecret = "test-client-secret"

	app := fiber.New()
	app.Get("/:provider/start", func(c fiber.Ctx) error { return h.Start(c) })

	now := time.Now()
	validClaims := jwt.MapClaims{
		"sub": "11111111-1111-1111-1111-111111111111", "org_id": "22222222-2222-2222-2222-222222222222",
		"type": "access", "iat": now.Unix(), "exp": now.Add(15 * time.Minute).Unix(),
	}

	cases := []struct {
		name  string
		token string
	}{
		{"wrong signing secret", signToken(t, "some-other-secret-entirely-different", validClaims)},
		{"refresh token instead of access", signToken(t, cfg.JWTSecret, jwt.MapClaims{
			"sub": "11111111-1111-1111-1111-111111111111", "type": "refresh",
			"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
		})},
		{"mfa_pending ticket instead of access", signToken(t, cfg.JWTSecret, jwt.MapClaims{
			"sub": "11111111-1111-1111-1111-111111111111", "type": "mfa_pending",
			"iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix(),
		})},
		{"expired token", signToken(t, cfg.JWTSecret, jwt.MapClaims{
			"sub": "11111111-1111-1111-1111-111111111111", "org_id": "22222222-2222-2222-2222-222222222222",
			"type": "access", "iat": now.Add(-time.Hour).Unix(), "exp": now.Add(-time.Minute).Unix(),
		})},
		{"malformed token", "not.a.jwt"},
		{"empty token", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := "/notion/start?token=" + url.QueryEscape(tc.token)
			if got := redirectError(t, app, target); got == "" {
				t.Error("expected a non-empty error redirect param")
			}
		})
	}
}

// TestOAuthStartAcceptsTokenSignedWithPreviousKey confirms the keyring
// fix: a token signed under a retired JWT_SECRET_PREVIOUS key must
// still pass verification here — the main API accepts these too, so
// this endpoint shouldn't be the one place that breaks for a user
// mid-session during a key rotation. Targets an unconfigured provider
// (no client id/secret set) specifically so a *valid* token still
// redirects with a distinguishable "oauth_not_configured" error rather
// than proceeding to a nil Redis client.
func TestOAuthStartAcceptsTokenSignedWithPreviousKey(t *testing.T) {
	retiredSecret := "retired-secret-that-is-at-least-32-chars"
	cfg := &config.Config{
		JWTSecret:         "current-secret-that-is-at-least-32-chars",
		JWTSecretPrevious: []string{retiredSecret},
		FrontendURL:       "https://app.example.com",
	}
	h := NewOAuthHandler(cfg, nil, nil, nil)

	app := fiber.New()
	app.Get("/:provider/start", func(c fiber.Ctx) error { return h.Start(c) })

	now := time.Now()
	token := signToken(t, retiredSecret, jwt.MapClaims{
		"sub": "11111111-1111-1111-1111-111111111111", "org_id": "22222222-2222-2222-2222-222222222222",
		"type": "access", "iat": now.Unix(), "exp": now.Add(15 * time.Minute).Unix(),
	})

	got := redirectError(t, app, "/apollo/start?token="+url.QueryEscape(token))
	if got != "oauth_not_configured" {
		t.Errorf("expected the retired-key-signed token to pass verification and fail only on "+
			"provider configuration, got error=%q", got)
	}
}
