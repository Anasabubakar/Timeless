package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/service"
)

type AutomationHandler struct {
	svc *service.AutomationService
}

func NewAutomationHandler(svc *service.AutomationService) *AutomationHandler {
	return &AutomationHandler{svc: svc}
}

func (h *AutomationHandler) List(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	automations, err := h.svc.List(c.Context(), orgID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list automations")
	}
	return c.JSON(fiber.Map{"data": automations})
}

func (h *AutomationHandler) Get(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	automation, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "automation not found")
	}
	return c.JSON(fiber.Map{"data": automation})
}

func (h *AutomationHandler) Create(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	var automation models.Automation
	if err := c.Bind().JSON(&automation); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	automation.OrganizationID = orgID
	automation.CreatedBy = &userID

	if err := h.svc.Create(c.Context(), &automation); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create automation")
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": automation})
}

func (h *AutomationHandler) Update(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	automation, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "automation not found")
	}

	if err := c.Bind().JSON(automation); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// Update() saves by primary key with no org check (see
	// repository.AutomationRepository.Update) — re-pin after the bind so
	// a client-supplied "organization_id" can't move this automation
	// into a different tenant's org.
	automation.OrganizationID = orgID
	automation.ID = id
	if err := h.svc.Update(c.Context(), automation); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update automation")
	}
	return c.JSON(fiber.Map{"data": automation})
}

func (h *AutomationHandler) Delete(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := h.svc.Delete(c.Context(), orgID, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete automation")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *AutomationHandler) Toggle(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var body struct {
		Active bool `json:"is_active"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.svc.ToggleActive(c.Context(), orgID, id, body.Active); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to toggle automation")
	}
	return c.JSON(fiber.Map{"success": true})
}
