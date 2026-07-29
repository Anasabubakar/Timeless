package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/pkg/reqbind"
	"github.com/timeless/backend/internal/service"
)

type AutomationHandler struct {
	svc *service.AutomationService
}

func NewAutomationHandler(svc *service.AutomationService) *AutomationHandler {
	return &AutomationHandler{svc: svc}
}

// AutomationInput is the client-writable subset of models.Automation.
// RunCount/LastRunAt/CreatedBy are deliberately excluded — they're
// bookkeeping the app itself maintains, not something a request body
// should be able to set (the previous direct-bind-into-the-model
// behavior would have silently accepted a client-forged run_count).
type AutomationInput struct {
	Name          string         `json:"name" validate:"required"`
	Description   *string        `json:"description,omitempty"`
	TriggerType   string         `json:"trigger_type" validate:"required"`
	TriggerConfig datatypes.JSON `json:"trigger_config"`
	Actions       datatypes.JSON `json:"actions"`
	Conditions    datatypes.JSON `json:"conditions"`
	IsActive      *bool          `json:"is_active,omitempty"`
}

func (in *AutomationInput) applyTo(automation *models.Automation) {
	automation.Name = in.Name
	automation.Description = in.Description
	automation.TriggerType = in.TriggerType
	if in.TriggerConfig != nil {
		automation.TriggerConfig = in.TriggerConfig
	}
	if in.Actions != nil {
		automation.Actions = in.Actions
	}
	if in.Conditions != nil {
		automation.Conditions = in.Conditions
	}
	if in.IsActive != nil {
		automation.IsActive = *in.IsActive
	}
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

	var input AutomationInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}

	automation := &models.Automation{OrganizationID: orgID, CreatedBy: &userID}
	input.applyTo(automation)

	if err := h.svc.Create(c.Context(), automation); err != nil {
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

	var input AutomationInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}
	input.applyTo(automation)

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

type toggleAutomationBody struct {
	Active bool `json:"is_active"`
}

func (h *AutomationHandler) Toggle(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var body toggleAutomationBody
	if verr := reqbind.JSON(c, &body); verr != nil {
		return verr
	}

	if err := h.svc.ToggleActive(c.Context(), orgID, id, body.Active); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to toggle automation")
	}
	return c.JSON(fiber.Map{"success": true})
}
