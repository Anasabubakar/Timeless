package router

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// TestEmptyPrefixGroupLeaksMiddlewareOntoLaterRoutes documents the bug
// that used to affect /ws's registration: an empty-prefix Group's
// middleware attaches as a fiber Use() at the parent's own prefix, which
// matches every route registered afterward at that prefix — regardless
// of which Group variable those later routes are added through. The
// production instance of this: `ws := app.Group("", authMw.HandleWS,
// ...); ws.Get("/ws", ...)` silently made every "/api/v1/notifications/*"
// and "/api/v1/events" route (registered after it) require a `token`
// query param instead of a normal Authorization header, since they
// inherited authMw.HandleWS (the WebSocket-specific auth path) instead
// of the real auth middleware.
func TestEmptyPrefixGroupLeaksMiddlewareOntoLaterRoutes(t *testing.T) {
	app := fiber.New()
	api := app.Group("/api/v1")

	var wsOnlyMiddlewareRanForNotifications bool
	wsOnlyMiddleware := func(c fiber.Ctx) error {
		if c.Path() != "/api/v1/ws" {
			wsOnlyMiddlewareRanForNotifications = true
		}
		return c.Next()
	}

	ws := api.Group("", wsOnlyMiddleware) // the bug: empty-prefix Group
	ws.Get("/ws", func(c fiber.Ctx) error { return c.SendString("ws") })

	// Registered afterward through a *different* Group variable — still
	// affected, because the middleware was attached at the shared prefix,
	// not to the /ws route specifically.
	api.Get("/notifications", func(c fiber.Ctx) error { return c.SendString("notifications") })

	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/notifications", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if !wsOnlyMiddlewareRanForNotifications {
		t.Fatal("expected this test to reproduce the leak (wsOnlyMiddleware running for /notifications) — " +
			"if it no longer does, fiber's Group semantics changed and the warning comment on router.go's " +
			"/ws registration may be stale")
	}
}

// TestDirectRouteAttachmentDoesNotLeak is the fix this app actually
// uses for /ws: middleware passed to app.Get(path, handler,
// middleware...) is scoped to that one route only, regardless of Go
// call-site ordering — fiber's App.register reorders variadic
// middleware args to run before the primary handler internally.
func TestDirectRouteAttachmentDoesNotLeak(t *testing.T) {
	app := fiber.New()
	api := app.Group("/api/v1")

	var wsOnlyMiddlewareRan bool
	wsOnlyMiddleware := func(c fiber.Ctx) error {
		wsOnlyMiddlewareRan = true
		return c.Next()
	}

	// Handler listed before middleware in the call, matching this app's
	// actual /ws registration — fiber still runs middleware first.
	api.Get("/ws", func(c fiber.Ctx) error { return c.SendString("ws") }, wsOnlyMiddleware)
	api.Get("/notifications", func(c fiber.Ctx) error { return c.SendString("notifications") })

	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/notifications", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if wsOnlyMiddlewareRan {
		t.Error("wsOnlyMiddleware ran for /notifications — it should be scoped to /ws only")
	}

	resp2, err := app.Test(httptest.NewRequest("GET", "/api/v1/ws", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp2.Body.Close()
	if !wsOnlyMiddlewareRan {
		t.Error("wsOnlyMiddleware did not run for /ws — it should have")
	}
}
