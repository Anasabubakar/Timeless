package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/pkg/reqbind"
	"github.com/timeless/backend/internal/service"
)

type ActivityHandler struct {
	svc *service.ActivityService
}

func NewActivityHandler(svc *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{svc: svc}
}

// ActivityInput is the client-writable subset for a manually-logged
// CRM note (e.g. "had a call with this contact"), distinct from the
// system-generated audit trail entries AuditLog middleware writes.
// Binding directly into the GORM model (the previous behavior) let a
// client set Type (the audit action verb — a client could forge a
// "deleted"/"permission_denied"/any system action name into their own
// entry, polluting or spoofing the audit trail) and IPAddress (spoofing
// where the "activity" came from). Both are now fixed server-side:
// Type is always "note" for manually-created entries, and IPAddress
// always comes from the request itself.
type ActivityInput struct {
	EntityType  string         `json:"entity_type" validate:"required"`
	EntityID    string         `json:"entity_id" validate:"required,uuid"`
	Subject     string         `json:"subject" validate:"required"`
	Description *string        `json:"description,omitempty"`
	Metadata    datatypes.JSON `json:"metadata"`
}

func (h *ActivityHandler) List(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	entityType := c.Query("entity_type")
	entityID := c.Query("entity_id")
	action := c.Query("type")

	activities, total, err := h.svc.List(c.Context(), orgID, limit, offset, entityType, entityID, action)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list activities")
	}

	return c.JSON(fiber.Map{
		"data":   activities,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *ActivityHandler) Create(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	var input ActivityInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}

	entityID, err := uuid.Parse(input.EntityID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid entity_id")
	}

	ip := c.IP()
	activity := &models.Activity{
		OrganizationID: orgID,
		UserID:         &userID,
		EntityType:     input.EntityType,
		EntityID:       entityID,
		Type:           "note", // manually-logged entries are never a forged system action
		Subject:        input.Subject,
		Description:    input.Description,
		Metadata:       input.Metadata,
		IPAddress:      &ip,
	}

	if err := h.svc.Create(c.Context(), activity); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create activity")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": activity})
}
