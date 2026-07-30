package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHealthHandler(db *gorm.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, rdb: rdb}
}

// pingTimeout bounds how long a readiness check waits on a dependency —
// a hung DB/Redis connection should fail the check quickly, not hang
// the probe (and whatever's waiting on it, e.g. a load balancer
// deciding whether to route traffic here) indefinitely.
const pingTimeout = 3 * time.Second

// Live is a liveness check: reports the process is up and serving
// requests, with no dependency checks. An orchestrator restarts the
// process if this stops responding — it should never fail just because
// the database is temporarily unreachable, that's what Ready is for.
func (h *HealthHandler) Live(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok", "service": "timeless-api"})
}

// Ready is a readiness check: reports whether this instance can
// actually serve traffic right now — DB and Redis both reachable. An
// orchestrator stops routing traffic here (without restarting the
// process) if this fails, which is the right response to "a dependency
// is down" as opposed to Live's "the process itself is broken."
func (h *HealthHandler) Ready(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), pingTimeout)
	defer cancel()

	checks := fiber.Map{}
	ready := true

	if sqlDB, err := h.db.DB(); err != nil || sqlDB.PingContext(ctx) != nil {
		checks["database"] = "unreachable"
		ready = false
	} else {
		checks["database"] = "ok"
	}

	if err := h.rdb.Ping(ctx).Err(); err != nil {
		checks["redis"] = "unreachable"
		ready = false
	} else {
		checks["redis"] = "ok"
	}

	status := fiber.StatusOK
	if !ready {
		status = fiber.StatusServiceUnavailable
	}

	return c.Status(status).JSON(fiber.Map{
		"status": map[bool]string{true: "ready", false: "not ready"}[ready],
		"checks": checks,
	})
}
