package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"

	"github.com/timeless/backend/internal/config"
)

// SecurityHeaders wraps fiber's helmet middleware with values chosen for
// a pure JSON API (no HTML/script ever served from here — the frontend
// is a separate origin): a maximally restrictive CSP since there's
// nothing here that should ever execute a script or load a frame, and
// Cross-Origin-Resource-Policy set to cross-origin rather than helmet's
// "same-origin" default, since the whole point of this API is being
// fetched from a different origin (the frontend) — same-origin CORP
// would have browsers block that legitimate cross-origin fetch.
func SecurityHeaders(cfg *config.Config) fiber.Handler {
	hstsMaxAge := 0
	if cfg.IsProduction() {
		// Only set HSTS when actually running behind HTTPS in
		// production — sending it in local dev over plain HTTP just
		// confuses browsers/tools for no benefit.
		hstsMaxAge = 31536000 // 1 year
	}

	return helmet.New(helmet.Config{
		ContentSecurityPolicy:     "default-src 'none'; frame-ancestors 'none'",
		XFrameOptions:             "DENY",
		ContentTypeNosniff:        "nosniff",
		ReferrerPolicy:            "no-referrer",
		PermissionPolicy:          "geolocation=(), microphone=(), camera=(), payment=(), usb=()",
		CrossOriginResourcePolicy: "cross-origin",
		CrossOriginOpenerPolicy:   "same-origin",
		HSTSMaxAge:                hstsMaxAge,
		// HSTSExcludeSubdomains defaults to false, i.e. subdomains ARE
		// included — the standard secure default once HSTS is enabled.
	})
}
