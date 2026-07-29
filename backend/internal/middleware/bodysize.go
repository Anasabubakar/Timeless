package middleware

import "github.com/gofiber/fiber/v3"

// MaxBodySize rejects a request whose declared Content-Length exceeds
// maxBytes before any body is read into memory. fiber.Config.BodyLimit
// (set to 10MB app-wide, sized for file uploads) is the only cap most
// routes had — a JSON CRUD endpoint has no legitimate reason to accept
// anywhere near that, and without a tighter per-route limit, a client
// could send a many-megabyte JSON body to an endpoint that was only
// ever meant to receive a few small fields, tying up a handler
// decoding/validating it for no reason.
func MaxBodySize(maxBytes int) fiber.Handler {
	return func(c fiber.Ctx) error {
		if n := c.Request().Header.ContentLength(); n > maxBytes {
			return fiber.NewError(fiber.StatusRequestEntityTooLarge, "request body too large")
		}
		return c.Next()
	}
}
