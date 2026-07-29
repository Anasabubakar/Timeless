package middleware

import "github.com/gofiber/fiber/v3"

// APIVersion is the current version of the /api/v1 surface. Bumping the
// URL prefix (a genuine v2) is the actual versioning mechanism; this
// header exists so clients and support tooling can see exactly which
// build of that surface answered a given request without it being tied
// to a deploy's git SHA (which isn't meaningful to an API consumer).
const APIVersion = "v1"

// WithAPIVersion sets X-API-Version on every response.
func WithAPIVersion(c fiber.Ctx) error {
	c.Set("X-API-Version", APIVersion)
	return c.Next()
}
