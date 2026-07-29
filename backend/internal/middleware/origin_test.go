package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestValidateOrigin(t *testing.T) {
	allowed := []string{"https://app.timeless.example", "https://staging.timeless.example"}

	cases := []struct {
		name       string
		method     string
		origin     string
		wantStatus int
	}{
		{"GET is never checked, even with a bad origin", "GET", "https://evil.example", 200},
		{"POST with no Origin header passes (server-to-server, curl, mobile)", "POST", "", 200},
		{"POST with an allowed origin passes", "POST", "https://app.timeless.example", 200},
		{"POST with a disallowed origin is rejected", "POST", "https://evil.example", 403},
		{"PATCH with a disallowed origin is rejected", "PATCH", "https://evil.example", 403},
		{"DELETE with a disallowed origin is rejected", "DELETE", "https://evil.example", 403},
		{"PUT with the second allowed origin passes", "PUT", "https://staging.timeless.example", 200},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Add([]string{tc.method}, "/x", func(c fiber.Ctx) error {
				return c.SendString("ok")
			}, ValidateOrigin(allowed))

			req := httptest.NewRequest(tc.method, "/x", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
		})
	}
}

func TestValidateOriginAlways(t *testing.T) {
	allowed := []string{"https://app.timeless.example"}

	cases := []struct {
		name       string
		origin     string
		wantStatus int
	}{
		{"GET with no Origin passes (WS upgrades from native clients)", "", 200},
		{"GET with an allowed origin passes", "https://app.timeless.example", 200},
		{"GET with a disallowed origin is rejected — unlike ValidateOrigin, GET is checked here", "https://evil.example", 403},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/ws", func(c fiber.Ctx) error {
				return c.SendString("ok")
			}, ValidateOriginAlways(allowed))

			req := httptest.NewRequest("GET", "/ws", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
		})
	}
}
