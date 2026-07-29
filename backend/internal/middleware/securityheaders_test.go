package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/timeless/backend/internal/config"
)

func TestSecurityHeadersSetsExpectedHeaders(t *testing.T) {
	cfg := &config.Config{Environment: "development"}
	app := fiber.New()
	app.Use(SecurityHeaders(cfg))
	app.Get("/x", func(c fiber.Ctx) error { return c.SendString("ok") })

	resp, err := app.Test(httptest.NewRequest("GET", "/x", nil))
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		header string
		want   string
	}{
		{"X-Frame-Options", "DENY"},
		{"X-Content-Type-Options", "nosniff"},
		{"Referrer-Policy", "no-referrer"},
		{"Cross-Origin-Resource-Policy", "cross-origin"},
	}
	for _, tc := range cases {
		got := resp.Header.Get(tc.header)
		if got != tc.want {
			t.Errorf("%s = %q, want %q", tc.header, got, tc.want)
		}
	}

	csp := resp.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src 'none'") {
		t.Errorf("Content-Security-Policy = %q, want it to contain default-src 'none'", csp)
	}
}

func TestSecurityHeadersOmitsHSTSOutsideProduction(t *testing.T) {
	cfg := &config.Config{Environment: "development"}
	app := fiber.New()
	app.Use(SecurityHeaders(cfg))
	app.Get("/x", func(c fiber.Ctx) error { return c.SendString("ok") })

	resp, err := app.Test(httptest.NewRequest("GET", "/x", nil))
	if err != nil {
		t.Fatal(err)
	}
	if hsts := resp.Header.Get("Strict-Transport-Security"); hsts != "" {
		t.Errorf("expected no HSTS header outside production, got %q", hsts)
	}
}

// helmet only ever sets Strict-Transport-Security when the request
// itself came in over HTTPS (checked independently of our config), so
// asserting the header's presence needs a real TLS round-trip through
// httptest — not worth the complexity here. hstsMaxAge is the pure
// piece of logic actually worth pinning down: is it wired to
// production correctly?
func TestHSTSMaxAge(t *testing.T) {
	cases := []struct {
		env  string
		want int
	}{
		{"production", 31536000},
		{"development", 0},
		{"staging", 0},
		{"", 0},
	}
	for _, tc := range cases {
		cfg := &config.Config{Environment: tc.env}
		if got := hstsMaxAge(cfg); got != tc.want {
			t.Errorf("hstsMaxAge(Environment=%q) = %d, want %d", tc.env, got, tc.want)
		}
	}
}
