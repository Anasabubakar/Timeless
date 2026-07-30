package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// TestHealthLiveHasNoDependencies confirms Live never touches db/rdb —
// it's constructed with both nil, which would panic on first use if the
// handler tried to check either. A liveness probe must report the
// process is up regardless of dependency health; that's what Ready is
// for. (Ready itself isn't covered here: it calls h.db.DB(), which
// panics on a nil *gorm.DB, so exercising it needs a real or mocked
// database connection this package doesn't have.)
func TestHealthLiveHasNoDependencies(t *testing.T) {
	h := NewHealthHandler(nil, nil)

	app := fiber.New()
	app.Get("/health/live", h.Live)

	resp, err := app.Test(httptest.NewRequest("GET", "/health/live", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}
