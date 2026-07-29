package middleware

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

// TestRequestLoggerOmitsQueryString guards against a future edit
// swapping ${path} for ${url}/${queries} (or logging headers) and
// silently putting a bearer token — OAuth start and the WebSocket
// upgrade both take one as a query param — into every request log line.
func TestRequestLoggerOmitsQueryString(t *testing.T) {
	var buf bytes.Buffer
	cfg := RequestLogger()
	cfg.Output = &buf

	app := fiber.New()
	app.Use(logger.New(cfg))
	app.Get("/integrations/notion/oauth/start", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	const secretToken = "eyJhbGciOiJIUzI1NiJ9.super-secret-token-value"
	req := httptest.NewRequest("GET", "/integrations/notion/oauth/start?token="+secretToken, nil)
	if _, err := app.Test(req); err != nil {
		t.Fatal(err)
	}

	logLine := buf.String()
	if logLine == "" {
		t.Fatal("expected a log line to be written")
	}
	if bytes.Contains([]byte(logLine), []byte(secretToken)) {
		t.Fatalf("request log line leaked the query-string token: %q", logLine)
	}
	if bytes.Contains([]byte(logLine), []byte("token=")) {
		t.Fatalf("request log line contains the query string at all: %q", logLine)
	}
}
