package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/timeless/backend/internal/integration"
	"github.com/timeless/backend/internal/logging"
	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/pkg/reqbind"
	"github.com/timeless/backend/internal/service"
)

type IntegrationHandler struct {
	svc *service.IntegrationService
}

func NewIntegrationHandler(svc *service.IntegrationService) *IntegrationHandler {
	return &IntegrationHandler{svc: svc}
}

func (h *IntegrationHandler) List(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	integrations, err := h.svc.List(c.Context(), orgID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list integrations")
	}
	return c.JSON(fiber.Map{"data": integrations})
}

func (h *IntegrationHandler) Get(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	integration, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "integration not found")
	}
	return c.JSON(fiber.Map{"data": integration})
}

func (h *IntegrationHandler) Create(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	var integration models.Integration
	if err := c.Bind().JSON(&integration); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	integration.OrganizationID = orgID
	integration.InstalledBy = &userID

	if err := h.svc.Create(c.Context(), &integration); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create integration")
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": integration})
}

func (h *IntegrationHandler) Update(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	integration, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "integration not found")
	}

	if err := c.Bind().JSON(integration); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.svc.Update(c.Context(), integration); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update integration")
	}
	return c.JSON(fiber.Map{"data": integration})
}

func (h *IntegrationHandler) Connect(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)
	provider := c.Params("provider")

	var input connectInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}

	rec, err := h.svc.Connect(c.Context(), orgID, userID, provider, service.ConnectInput{Credentials: input.Credentials})
	if err != nil {
		// client.Validate's error can carry the provider's raw HTTP
		// response body (e.g. a rejected-credential response) — log it
		// server-side, don't relay it into the API response.
		logging.Printf("integration: connect failed for org %s provider %s: %v", orgID, provider, err)
		return fiber.NewError(fiber.StatusBadRequest, "could not connect "+provider+" — check the credentials and try again")
	}
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"data": rec})
}

type connectInput struct {
	Credentials map[string]string `json:"credentials" validate:"required,min=1"`
}

func (h *IntegrationHandler) Delete(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := h.svc.Delete(c.Context(), orgID, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete integration")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// EnableInboundWebhook returns the org's inbound webhook URL for a
// provider that has no OAuth/signing flow of its own (Zapier), generating
// an unguessable token on first call. The URL itself is the secret, so
// callers should treat this response like a credential — it's never
// re-derivable from the dashboard's normal integration listing.
func (h *IntegrationHandler) EnableInboundWebhook(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	provider := c.Params("provider")

	rec, err := h.svc.EnsureInboundWebhookToken(c.Context(), orgID, provider)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"webhook_url": rec.WebhookURL})
}

// TriggerSync manually enqueues a sync for this integration right now,
// instead of waiting for the next webhook/scheduled poll.
func (h *IntegrationHandler) TriggerSync(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := h.svc.TriggerSync(c.Context(), orgID, userID, id); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.SendStatus(fiber.StatusAccepted)
}

// Revoke disconnects an integration but keeps its sync history, so the
// dashboard can still show "revoked 2 days ago" instead of the row just
// disappearing.
func (h *IntegrationHandler) Revoke(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	provider, err := h.svc.Revoke(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to revoke integration")
	}

	// Neither Notion's nor Apollo's OAuth API documents a token
	// revocation endpoint — this call only wipes Timeless's own copy of
	// the credentials. Say so, rather than letting "revoked" imply a
	// stronger guarantee (the token invalidated everywhere) than what
	// actually happened.
	return c.JSON(fiber.Map{
		"message": "integration disconnected from Timeless — to fully revoke access, also remove it from your " + provider + " account's connected apps/integrations settings",
	})
}

// Dashboard returns connection health, sync history, and job status for
// every integration in the org.
func (h *IntegrationHandler) Dashboard(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	entries, err := h.svc.Dashboard(c.Context(), orgID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load integration dashboard")
	}
	return c.JSON(fiber.Map{"data": entries})
}

// SyncConflicts returns every record across every integration currently
// awaiting conflict resolution — both sides changed since the last sync,
// so nothing was applied automatically.
func (h *IntegrationHandler) SyncConflicts(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	conflicts, err := h.svc.ConflictQueue(c.Context(), orgID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load conflict queue")
	}
	return c.JSON(fiber.Map{"data": conflicts})
}

// SyncActivity returns the org's most recent sync actions (pushed/pulled/
// conflict detected/failed) across every entity and integration — the
// Sync Dashboard's live activity feed.
func (h *IntegrationHandler) SyncActivity(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	activity, err := h.svc.RecentSyncActivity(c.Context(), orgID, limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load sync activity")
	}
	return c.JSON(fiber.Map{"data": activity})
}

// ZapierApps returns the third-party apps discovered through the org's
// Zapier connection.
func (h *IntegrationHandler) ZapierApps(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	apps, agentic, err := h.svc.DiscoverConnectedApps(c.Context(), orgID)
	if err != nil {
		// Wraps a live call to Zapier's MCP endpoint — the error can
		// carry Zapier's raw response.
		logging.Printf("integration: zapier app discovery failed for org %s: %v", orgID, err)
		return fiber.NewError(fiber.StatusBadGateway, "could not reach Zapier — try again shortly")
	}
	return c.JSON(fiber.Map{"data": apps, "agentic_mode": agentic})
}

// RotateCredentials re-encrypts every stored credential in the org that
// isn't already under the current encryption key.
func (h *IntegrationHandler) RotateCredentials(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	result, err := h.svc.RotateCredentials(c.Context(), orgID)
	if err != nil {
		logging.Printf("integration: credential rotation failed for org %s: %v", orgID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "credential rotation failed")
	}
	return c.JSON(fiber.Map{"data": result})
}

type pushNotionPageRequest struct {
	Properties             map[string]interface{} `json:"properties"`
	ExpectedLastEditedTime string                 `json:"expected_last_edited_time"`
}

// PushNotionPage writes SponsorOS-side changes back to a Notion page.
// Returns 409 Conflict (not 500) when Notion's copy changed since we last
// read it, so the frontend can show "this was edited in Notion, refresh
// and retry" instead of a generic error.
func (h *IntegrationHandler) PushNotionPage(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	pageID := c.Params("pageID")

	var req pushNotionPageRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	err := h.svc.PushToNotionPage(c.Context(), orgID, pageID, req.Properties, req.ExpectedLastEditedTime)
	if err != nil {
		var conflictErr *integration.ConflictError
		if errors.As(err, &conflictErr) {
			return fiber.NewError(fiber.StatusConflict, conflictErr.Error())
		}
		// Anything else here wraps a live Notion API call — the error
		// can carry Notion's raw response.
		logging.Printf("integration: push to notion page %s failed for org %s: %v", pageID, orgID, err)
		return fiber.NewError(fiber.StatusBadGateway, "could not update the Notion page — try again shortly")
	}
	return c.SendStatus(fiber.StatusNoContent)
}
