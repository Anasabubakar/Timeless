package middleware

import (
	"slices"

	"github.com/gofiber/fiber/v3"
)

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
//
// A request with NO Origin header at all is allowed through — that's
// the normal shape for server-to-server calls, mobile apps, and tools
// like curl/Postman, none of which are a CSRF-relevant threat model
// (this API has no cookie-based session for a browser to
// forge-and-auto-attach in the first place; auth is a bearer token a
// forged cross-origin request would never have access to).
func ValidateOrigin(allowedOrigins []string) fiber.Handler {
	return func(c fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete:
		default:
			return c.Next()
		}

		origin := c.Get(fiber.HeaderOrigin)
		if origin == "" {
			return c.Next()
		}

		if !slices.Contains(allowedOrigins, origin) {
			return fiber.NewError(fiber.StatusForbidden, "request origin not allowed")
		}

		return c.Next()
	}
}
