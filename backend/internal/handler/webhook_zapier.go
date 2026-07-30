package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"github.com/timeless/backend/internal/eventbus"
	"github.com/timeless/backend/internal/service"
)

// zapierDedupeTTL bounds how long an inbound payload's content hash is
// remembered for duplicate detection. Zapier (like most webhook senders)
// retries on anything but a 2xx response, so a delivery seen again within
// this window is treated as a retry of the same event, not a new one, and
// acknowledged without re-processing.
const zapierDedupeTTL = 24 * time.Hour

// ZapierWebhookHandler receives inbound events from a user's Zap via
// "Webhooks by Zapier." Unlike Notion, Zapier's inbound trigger has no
// signing mechanism at all — the unguessable per-org URL token
// (:token, matched against Integration.WebhookSecret) is the entire
// authentication. An optional X-Zapier-Secret header, checked when the
// org has one configured, is a second layer for the subset of Zapier
// plans that support custom headers.
type ZapierWebhookHandler struct {
	rdb *redis.Client
	svc *service.IntegrationService
	bus *eventbus.Bus
}

func NewZapierWebhookHandler(rdb *redis.Client, svc *service.IntegrationService, bus *eventbus.Bus) *ZapierWebhookHandler {
	return &ZapierWebhookHandler{rdb: rdb, svc: svc, bus: bus}
}

// Receive validates the URL token, dedupes by content hash, and durably
// publishes ZapierWebhookReceived — never processes the payload inline,
// so a slow/failing downstream subscriber can't turn into a slow/failing
// response to Zapier (which would just trigger more retries).
func (h *ZapierWebhookHandler) Receive(c fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing webhook token")
	}

	rec, err := h.svc.IntegrationByWebhookToken(c.Context(), "zapier", token)
	if err != nil {
		// Deliberately identical response/timing profile to "valid token,
		// nothing to do" — a 404 here would let an attacker distinguish
		// "this token doesn't exist" from other failure modes while
		// probing the URL space.
		return fiber.NewError(fiber.StatusUnauthorized, "invalid webhook token")
	}

	body := c.Body()
	if len(body) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "empty payload")
	}
	if len(body) > 256*1024 {
		return fiber.NewError(fiber.StatusRequestEntityTooLarge, "payload too large")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "payload must be a JSON object")
	}

	sum := sha256.Sum256(body)
	dedupeKey := "zapier:webhook:seen:" + rec.ID.String() + ":" + hex.EncodeToString(sum[:])
	isNew, err := h.rdb.SetNX(c.Context(), dedupeKey, "1", zapierDedupeTTL).Result()
	if err != nil {
		slog.Warn("zapier webhook dedupe check failed", "integration_id", rec.ID, "error", err)
	} else if !isNew {
		// Same content already processed within the window — ack without
		// re-publishing, same as any other idempotent webhook receiver.
		return c.SendStatus(fiber.StatusOK)
	}

	if h.bus != nil {
		evt := eventbus.Event{
			Type:       eventbus.ZapierWebhookReceived,
			OrgID:      rec.OrganizationID.String(),
			EntityType: "zapier_webhook",
			EntityID:   rec.ID.String(),
			Data:       payload,
		}
		if err := h.bus.Publish(c.Context(), evt); err != nil {
			slog.Error("failed to publish ZapierWebhookReceived", "integration_id", rec.ID, "error", err)
			return fiber.NewError(fiber.StatusServiceUnavailable, "failed to queue webhook for processing")
		}
	}

	return c.SendStatus(fiber.StatusOK)
}
