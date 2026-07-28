package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
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

	var input service.ConnectInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	rec, err := h.svc.Connect(c.Context(), orgID, userID, provider, input)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"data": rec})
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

// Revoke disconnects an integration but keeps its sync history, so the
// dashboard can still show "revoked 2 days ago" instead of the row just
// disappearing.
func (h *IntegrationHandler) Revoke(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := h.svc.Revoke(c.Context(), orgID, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to revoke integration")
	}
	return c.SendStatus(fiber.StatusNoContent)
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

// ZapierApps returns the third-party apps discovered through the org's
// Zapier connection.
func (h *IntegrationHandler) ZapierApps(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	apps, agentic, err := h.svc.DiscoverConnectedApps(c.Context(), orgID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"data": apps, "agentic_mode": agentic})
}
