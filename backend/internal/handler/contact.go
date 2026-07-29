package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/pkg/reqbind"
	"github.com/timeless/backend/internal/service"
)

type ContactHandler struct {
	svc *service.ContactService
}

func NewContactHandler(svc *service.ContactService) *ContactHandler {
	return &ContactHandler{svc: svc}
}

// ContactInput is the client-writable subset of models.Contact —
// binding directly into the GORM model (the previous behavior) let a
// client set their own ID/OrganizationID/CreatedAt on create, the same
// mass-assignment gap fixed for CompanyInput/SponsorInput.
type ContactInput struct {
	CompanyID       *uuid.UUID     `json:"company_id,omitempty"`
	FirstName       string         `json:"first_name" validate:"required"`
	LastName        string         `json:"last_name" validate:"required"`
	Email           *string        `json:"email,omitempty" validate:"omitempty,email"`
	Phone           *string        `json:"phone,omitempty"`
	Title           *string        `json:"title,omitempty"`
	Department      *string        `json:"department,omitempty"`
	LinkedinURL     *string        `json:"linkedin_url,omitempty" validate:"omitempty,url"`
	AvatarURL       *string        `json:"avatar_url,omitempty" validate:"omitempty,url"`
	Notes           *string        `json:"notes,omitempty"`
	Tags            pq.StringArray `json:"tags"`
	CustomFields    datatypes.JSON `json:"custom_fields"`
	LastContactedAt *time.Time     `json:"last_contacted_at,omitempty"`
	Status          string         `json:"status"`
}

func (in *ContactInput) applyTo(contact *models.Contact) {
	contact.CompanyID = in.CompanyID
	contact.FirstName = in.FirstName
	contact.LastName = in.LastName
	contact.Email = in.Email
	contact.Phone = in.Phone
	contact.Title = in.Title
	contact.Department = in.Department
	contact.LinkedinURL = in.LinkedinURL
	contact.AvatarURL = in.AvatarURL
	contact.Notes = in.Notes
	contact.Tags = in.Tags
	contact.CustomFields = in.CustomFields
	contact.LastContactedAt = in.LastContactedAt
	if in.Status != "" {
		contact.Status = in.Status
	}
}

func (h *ContactHandler) List(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	search := c.Query("search")

	contacts, total, err := h.svc.List(c.Context(), orgID, limit, offset, search)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list contacts")
	}

	return c.JSON(fiber.Map{
		"data":   contacts,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *ContactHandler) Get(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	contact, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "contact not found")
	}

	return c.JSON(fiber.Map{"data": contact})
}

func (h *ContactHandler) Create(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var input ContactInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}

	contact := &models.Contact{OrganizationID: orgID}
	input.applyTo(contact)

	if err := h.svc.Create(c.Context(), contact); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create contact")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": contact})
}

func (h *ContactHandler) Update(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	contact, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "contact not found")
	}

	var input ContactInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}
	input.applyTo(contact)

	contact.OrganizationID = orgID
	contact.ID = id
	if err := h.svc.Update(c.Context(), contact); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update contact")
	}

	return c.JSON(fiber.Map{"data": contact})
}

func (h *ContactHandler) Delete(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := h.svc.Delete(c.Context(), orgID, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete contact")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
