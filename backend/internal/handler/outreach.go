package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/sponsoros/backend/internal/middleware"
	"github.com/sponsoros/backend/internal/models"
	"github.com/sponsoros/backend/internal/service"
)

type OutreachHandler struct {
	svc *service.OutreachService
}

func NewOutreachHandler(svc *service.OutreachService) *OutreachHandler {
	return &OutreachHandler{svc: svc}
}

func (h *OutreachHandler) ListSequences(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	status := c.Query("status")

	sequences, err := h.svc.ListSequences(c.Context(), orgID, status)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list sequences")
	}

	return c.JSON(fiber.Map{"data": sequences})
}

func (h *OutreachHandler) GetSequence(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	seq, err := h.svc.GetSequence(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "sequence not found")
	}

	return c.JSON(fiber.Map{"data": seq})
}

func (h *OutreachHandler) CreateSequence(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	var seq models.OutreachSequence
	if err := c.Bind().JSON(&seq); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	seq.OrganizationID = orgID
	seq.CreatedBy = &userID
	if err := h.svc.CreateSequence(c.Context(), &seq); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create sequence")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": seq})
}

func (h *OutreachHandler) UpdateSequence(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	seq, err := h.svc.GetSequence(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "sequence not found")
	}

	if err := c.Bind().JSON(seq); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.svc.UpdateSequence(c.Context(), seq); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update sequence")
	}

	return c.JSON(fiber.Map{"data": seq})
}

func (h *OutreachHandler) DeleteSequence(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := h.svc.DeleteSequence(c.Context(), orgID, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete sequence")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *OutreachHandler) Enroll(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var enrollment models.Enrollment
	if err := c.Bind().JSON(&enrollment); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	enrollment.OrganizationID = orgID
	if err := h.svc.CreateEnrollment(c.Context(), &enrollment); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to enroll contact")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": enrollment})
}
