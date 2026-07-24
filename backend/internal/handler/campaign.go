package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/service"
)

type CampaignHandler struct {
	svc *service.CampaignService
}

func NewCampaignHandler(svc *service.CampaignService) *CampaignHandler {
	return &CampaignHandler{svc: svc}
}

func (h *CampaignHandler) List(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	campaigns, total, err := h.svc.List(c.Context(), orgID, limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list campaigns")
	}

	return c.JSON(fiber.Map{
		"campaigns": campaigns,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

func (h *CampaignHandler) Create(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var campaign models.Campaign
	if err := c.Bind().JSON(&campaign); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	campaign.OrganizationID = orgID
	if err := h.svc.Create(c.Context(), &campaign); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create campaign")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"campaign": campaign})
}

func (h *CampaignHandler) Get(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid campaign id")
	}

	campaign, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "campaign not found")
	}

	return c.JSON(fiber.Map{"campaign": campaign})
}

func (h *CampaignHandler) Update(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid campaign id")
	}

	campaign, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "campaign not found")
	}

	if err := c.Bind().JSON(campaign); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	campaign.OrganizationID = orgID
	campaign.ID = id
	if err := h.svc.Update(c.Context(), campaign); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update campaign")
	}

	return c.JSON(fiber.Map{"campaign": campaign})
}

func (h *CampaignHandler) Delete(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid campaign id")
	}

	if err := h.svc.Delete(c.Context(), orgID, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete campaign")
	}

	return c.JSON(fiber.Map{"message": "campaign deleted"})
}
