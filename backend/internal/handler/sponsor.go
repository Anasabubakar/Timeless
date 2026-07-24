package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/service"
)

type SponsorHandler struct {
	svc *service.SponsorService
}

func NewSponsorHandler(svc *service.SponsorService) *SponsorHandler {
	return &SponsorHandler{svc: svc}
}

func (h *SponsorHandler) List(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	stage := c.Query("stage")

	var campaignID *uuid.UUID
	if cid := c.Query("campaign_id"); cid != "" {
		parsed, err := uuid.Parse(cid)
		if err == nil {
			campaignID = &parsed
		}
	}

	sponsors, total, err := h.svc.List(c.Context(), orgID, limit, offset, campaignID, stage)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list sponsors")
	}

	return c.JSON(fiber.Map{
		"sponsors": sponsors,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

func (h *SponsorHandler) Create(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var sponsor models.Sponsor
	if err := c.Bind().JSON(&sponsor); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	sponsor.OrganizationID = orgID
	if err := h.svc.Create(c.Context(), &sponsor); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create sponsor")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"sponsor": sponsor})
}

func (h *SponsorHandler) Get(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid sponsor id")
	}

	sponsor, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "sponsor not found")
	}

	return c.JSON(fiber.Map{"sponsor": sponsor})
}

func (h *SponsorHandler) Update(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid sponsor id")
	}

	sponsor, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "sponsor not found")
	}

	if err := c.Bind().JSON(sponsor); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	sponsor.OrganizationID = orgID
	sponsor.ID = id
	if err := h.svc.Update(c.Context(), sponsor); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update sponsor")
	}

	return c.JSON(fiber.Map{"sponsor": sponsor})
}

func (h *SponsorHandler) Delete(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid sponsor id")
	}

	if err := h.svc.Delete(c.Context(), orgID, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete sponsor")
	}

	return c.JSON(fiber.Map{"message": "sponsor deleted"})
}

func (h *SponsorHandler) UpdateStage(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid sponsor id")
	}

	var body struct {
		Stage    string `json:"stage"`
		Position int    `json:"position"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Stage == "" {
		return fiber.NewError(fiber.StatusBadRequest, "stage is required")
	}

	if err := h.svc.UpdateStage(c.Context(), orgID, id, body.Stage, body.Position); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update stage")
	}

	return c.JSON(fiber.Map{"message": "stage updated"})
}
