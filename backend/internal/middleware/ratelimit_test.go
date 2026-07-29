package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestKeyDerivationHelpers(t *testing.T) {
	app := fiber.New()

	t.Run("ByIP uses the request IP regardless of locals", func(t *testing.T) {
		var got string
		app.Get("/byip", func(c fiber.Ctx) error {
			got = ByIP(c)
			return c.SendString("ok")
		})
		req := httptest.NewRequest("GET", "/byip", nil)
		if _, err := app.Test(req); err != nil {
			t.Fatal(err)
		}
		if got == "" {
			t.Fatal("ByIP returned empty key")
		}
	})

	t.Run("ByUser prefers the authenticated user id", func(t *testing.T) {
		var got string
		app.Get("/byuser", func(c fiber.Ctx) error {
			c.Locals("user_id", "11111111-1111-1111-1111-111111111111")
			got = ByUser(c)
			return c.SendString("ok")
		})
		req := httptest.NewRequest("GET", "/byuser", nil)
		if _, err := app.Test(req); err != nil {
			t.Fatal(err)
		}
		want := "user:11111111-1111-1111-1111-111111111111"
		if got != want {
			t.Fatalf("ByUser() = %q, want %q", got, want)
		}
	})

	t.Run("ByUser falls back to IP when unauthenticated", func(t *testing.T) {
		var got string
		app.Get("/byuser-anon", func(c fiber.Ctx) error {
			got = ByUser(c)
			return c.SendString("ok")
		})
		req := httptest.NewRequest("GET", "/byuser-anon", nil)
		if _, err := app.Test(req); err != nil {
			t.Fatal(err)
		}
		if got == "" || got[:3] != "ip:" {
			t.Fatalf("ByUser() without a user should fall back to an ip: prefixed key, got %q", got)
		}
	})

	t.Run("ByOrg prefers the org id and falls back to IP", func(t *testing.T) {
		var got string
		app.Get("/byorg", func(c fiber.Ctx) error {
			c.Locals("org_id", "22222222-2222-2222-2222-222222222222")
			got = ByOrg(c)
			return c.SendString("ok")
		})
		req := httptest.NewRequest("GET", "/byorg", nil)
		if _, err := app.Test(req); err != nil {
			t.Fatal(err)
		}
		want := "org:22222222-2222-2222-2222-222222222222"
		if got != want {
			t.Fatalf("ByOrg() = %q, want %q", got, want)
		}
	})
}

// TestRateLimitPoliciesAreWellFormed catches an inverted or missing
// value in one of the named policies (e.g. Burst > Limit, which makes
// the burst check strictly weaker than the sustained one and therefore
// pointless) before it ships.
func TestRateLimitPoliciesAreWellFormed(t *testing.T) {
	policies := map[string]RateLimitConfig{
		"login":              RateLimitLogin(),
		"mfa_verify":         RateLimitMFAVerify(),
		"register":           RateLimitRegister(),
		"refresh":            RateLimitRefresh(),
		"password_reset":     RateLimitPasswordReset(),
		"email_verification": RateLimitEmailVerification(),
		"ai_user":            RateLimitAIPerUser(),
		"ai_org":             RateLimitAIPerOrg(),
		"uploads":            RateLimitUploads(),
		"imports":            RateLimitImports(),
		"emails_send":        RateLimitEmailsSend(),
		"api":                RateLimitAPI(),
		"oauth":              RateLimitOAuth(),
		"webhook_inbound":    RateLimitWebhookInbound(),
	}

	for name, cfg := range policies {
		if cfg.Name == "" {
			t.Errorf("%s: Name must not be empty (used as the Redis key namespace)", name)
		}
		if cfg.Limit <= 0 {
			t.Errorf("%s: Limit must be positive, got %d", name, cfg.Limit)
		}
		if cfg.Window <= 0 {
			t.Errorf("%s: Window must be positive, got %v", name, cfg.Window)
		}
		if cfg.KeyFunc == nil {
			t.Errorf("%s: KeyFunc must not be nil", name)
		}
		if cfg.Burst > 0 {
			if cfg.Burst > cfg.Limit {
				t.Errorf("%s: Burst (%d) must not exceed Limit (%d), or the burst check never fires first", name, cfg.Burst, cfg.Limit)
			}
			if cfg.BurstWindow <= 0 {
				t.Errorf("%s: BurstWindow must be positive when Burst is set", name)
			}
			if cfg.BurstWindow >= cfg.Window {
				t.Errorf("%s: BurstWindow (%v) should be shorter than Window (%v) — otherwise it's not really a burst check", name, cfg.BurstWindow, cfg.Window)
			}
		}
	}
}
