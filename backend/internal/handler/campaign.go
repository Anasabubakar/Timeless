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

type CampaignHandler struct {
	svc *service.CampaignService
}

func NewCampaignHandler(svc *service.CampaignService) *CampaignHandler {
	return &CampaignHandler{svc: svc}
}

// CampaignInput is the client-writable subset of models.Campaign.
// CreatedBy is deliberately excluded — it's set from the authenticated
// user, never accepted from the request body, unlike the previous
// direct-bind-into-the-model behavior which would have silently
// accepted (and let a client forge) any created_by a request supplied.
type CampaignInput struct {
	ProjectID      uuid.UUID      `json:"project_id" validate:"required"`
	Name           string         `json:"name" validate:"required"`
	Description    *string        `json:"description,omitempty"`
	Status         string         `json:"status"`
	GoalAmount     *float64       `json:"goal_amount,omitempty"`
	RaisedAmount   float64        `json:"raised_amount"`
	Currency       string         `json:"currency"`
	StartDate      *string        `json:"start_date,omitempty"`
	EndDate        *string        `json:"end_date,omitempty"`
	PipelineStages datatypes.JSON `json:"pipeline_stages"`
	Settings       datatypes.JSON `json:"settings"`
}

func (in *CampaignInput) applyTo(campaign *models.Campaign) {
	campaign.ProjectID = in.ProjectID
	campaign.Name = in.Name
	campaign.Description = in.Description
	if in.Status != "" {
		campaign.Status = in.Status
	}
	campaign.GoalAmount = in.GoalAmount
	campaign.RaisedAmount = in.RaisedAmount
	if in.Currency != "" {
		campaign.Currency = in.Currency
	}
	campaign.StartDate = in.StartDate
	campaign.EndDate = in.EndDate
	if in.PipelineStages != nil {
		campaign.PipelineStages = in.PipelineStages
	}
	if in.Settings != nil {
		campaign.Settings = in.Settings
	}
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
	userID := middleware.GetUserID(c)

	var input CampaignInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}

	campaign := &models.Campaign{OrganizationID: orgID, CreatedBy: &userID}
	input.applyTo(campaign)

	if err := h.svc.Create(c.Context(), campaign); err != nil {
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

	var input CampaignInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}
	input.applyTo(campaign)

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
