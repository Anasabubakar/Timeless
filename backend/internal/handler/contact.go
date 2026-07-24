package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/service"
)

type ContactHandler struct {
	svc *service.ContactService
}

func NewContactHandler(svc *service.ContactService) *ContactHandler {
	return &ContactHandler{svc: svc}
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
		"data":  contacts,
		"total": total,
		"limit": limit,
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

	var contact models.Contact
	if err := c.Bind().JSON(&contact); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	contact.OrganizationID = orgID
	if err := h.svc.Create(c.Context(), &contact); err != nil {
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

	if err := c.Bind().JSON(contact); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

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
