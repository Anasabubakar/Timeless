package middleware

import (
	"slices"

	"github.com/gofiber/fiber/v3"
)

// checkOrigin is the shared decision: a present-but-disallowed Origin
// is rejected; a missing Origin (server-to-server calls, mobile apps,
// curl/Postman — none of which are a CSRF-relevant threat model, since
// this API has no cookie-based session for a browser to
// forge-and-auto-attach and auth is a bearer token a forged
// cross-origin request would never have access to) is allowed through.
func checkOrigin(c fiber.Ctx, allowedOrigins []string) error {
	origin := c.Get(fiber.HeaderOrigin)
	if origin == "" || slices.Contains(allowedOrigins, origin) {
		return nil
	}
	return fiber.NewError(fiber.StatusForbidden, "request origin not allowed")
}

// ValidateOrigin rejects state-changing (POST/PUT/PATCH/DELETE)
// requests whose Origin header is present but doesn't match one of the
// configured allowed origins. This is deliberately independent of CORS:
// CORS is enforced client-side by the browser (a fetch() from a
// disallowed origin fails before the app ever sees a response), so it
// only ever protects against a properly-behaving browser respecting
// CORS at all — it does nothing against a CORS misconfiguration, an
// older browser, or a non-browser HTTP client crafting the header
// itself. This gives the server its own opinion instead of trusting the
// browser to have enforced one.
func ValidateOrigin(allowedOrigins []string) fiber.Handler {
	return func(c fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete:
		default:
			return c.Next()
		}
		if err := checkOrigin(c, allowedOrigins); err != nil {
			return err
		}
		return c.Next()
	}
}

// ValidateOriginAlways is ValidateOrigin without the method allowlist —
// for routes like the WebSocket upgrade, which is always a GET but
// establishes a stateful, long-lived connection unlike a normal safe
// GET, so it doesn't fit ValidateOrigin's "only check state-changing
// methods" scoping.
func ValidateOriginAlways(allowedOrigins []string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if err := checkOrigin(c, allowedOrigins); err != nil {
			return err
		}
		return c.Next()
	}
}
