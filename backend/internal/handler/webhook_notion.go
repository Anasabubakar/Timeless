package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"github.com/timeless/backend/internal/eventbus"
	"github.com/timeless/backend/internal/service"
)

// notionWebhookTokenKey stores the verification_token Notion hands us once,
// the first time this URL is saved in the integration's Webhooks
// configuration tab. It's app-wide (one Notion OAuth app, one webhook
// endpoint), not per-org, so a single Redis key is the right scope.
const notionWebhookTokenKey = "notion:webhook:verification_token"

// NotionWebhookHandler receives real-time change events from Notion
// (developers.notion.com/reference/webhooks): a one-time verification
// handshake, then signed events for every subsequent workspace change.
type NotionWebhookHandler struct {
	rdb *redis.Client
	svc *service.IntegrationService
	bus *eventbus.Bus
}

func NewNotionWebhookHandler(rdb *redis.Client, svc *service.IntegrationService, bus *eventbus.Bus) *NotionWebhookHandler {
	return &NotionWebhookHandler{rdb: rdb, svc: svc, bus: bus}
}

type notionVerificationPing struct {
	VerificationToken string `json:"verification_token"`
}

type notionWebhookEvent struct {
	WorkspaceID string `json:"workspace_id"`
	Type        string `json:"type"`
	Entity      struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"entity"`
}

// Receive handles both the one-time verification handshake and every
// subsequent signed event. It always returns quickly (enqueueing a sync
// rather than processing inline) so Notion never sees a slow/failing
// endpoint and disables the webhook.
func (h *NotionWebhookHandler) Receive(c fiber.Ctx) error {
	body := c.Body()

	var ping notionVerificationPing
	if err := json.Unmarshal(body, &ping); err == nil && ping.VerificationToken != "" {
		if err := h.rdb.Set(c.Context(), notionWebhookTokenKey, ping.VerificationToken, 0).Err(); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to store verification token")
		}
		return c.SendStatus(fiber.StatusOK)
	}

	token, err := h.rdb.Get(c.Context(), notionWebhookTokenKey).Result()
	if err != nil || token == "" {
		// The webhook was hit before the verification handshake completed
		// (or the token was never persisted) — nothing to verify against yet.
		return fiber.NewError(fiber.StatusPreconditionFailed, "notion webhook not yet verified")
	}

	mac := hmac.New(sha256.New, []byte(token))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(c.Get("X-Notion-Signature"))) {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid notion webhook signature")
	}

	var event notionWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil || event.WorkspaceID == "" {
		return c.SendStatus(fiber.StatusOK)
	}

	// Best-effort routing: if we can't match this workspace to a connected
	// integration (e.g. it was disconnected), still ack 200 — there's
	// nothing actionable to retry, and Notion isn't at fault.
	orgID, _, err := h.svc.EnqueueWebhookSync(c.Context(), "notion", event.WorkspaceID)

	// The generic sync above is the coarse fallback (full incremental
	// resync, eventually catches everything); when the event names a
	// specific page, also publish a targeted pull so the mapping-engine
	// sync pipeline converges immediately instead of waiting for the
	// next resync.
	if err == nil && h.bus != nil && event.Entity.Type == "page" && event.Entity.ID != "" {
		_ = h.bus.Publish(c.Context(), eventbus.Event{
			Type:  eventbus.NotionChanged,
			OrgID: orgID.String(),
			Data: map[string]interface{}{
				"external_system": "notion",
				"external_id":     event.Entity.ID,
			},
		})
	}

	return c.SendStatus(fiber.StatusOK)
}
