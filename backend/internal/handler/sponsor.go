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

type SponsorHandler struct {
	svc *service.SponsorService
}

func NewSponsorHandler(svc *service.SponsorService) *SponsorHandler {
	return &SponsorHandler{svc: svc}
}

// SponsorInput is the client-writable subset of models.Sponsor —
// binding directly into the GORM model (the previous behavior) let a
// client set their own ID/CreatedAt/UpdatedAt on create, the same
// mass-assignment gap fixed for CompanyInput in an earlier phase.
type SponsorInput struct {
	CampaignID       uuid.UUID      `json:"campaign_id" validate:"required"`
	CompanyID        uuid.UUID      `json:"company_id" validate:"required"`
	Stage            string         `json:"stage"`
	DealValue        *float64       `json:"deal_value,omitempty"`
	Probability      *int           `json:"probability,omitempty"`
	Tier             *string        `json:"tier,omitempty"`
	AssignedTo       *uuid.UUID     `json:"assigned_to,omitempty"`
	PrimaryContactID *uuid.UUID     `json:"primary_contact_id,omitempty"`
	ExpectedClose    *string        `json:"expected_close,omitempty"`
	ActualClose      *string        `json:"actual_close,omitempty"`
	LostReason       *string        `json:"lost_reason,omitempty"`
	Notes            *string        `json:"notes,omitempty"`
	CustomFields     datatypes.JSON `json:"custom_fields"`
	Position         int            `json:"position"`
}

func (in *SponsorInput) applyTo(sponsor *models.Sponsor) {
	sponsor.CampaignID = in.CampaignID
	sponsor.CompanyID = in.CompanyID
	if in.Stage != "" {
		sponsor.Stage = in.Stage
	}
	sponsor.DealValue = in.DealValue
	sponsor.Probability = in.Probability
	sponsor.Tier = in.Tier
	sponsor.AssignedTo = in.AssignedTo
	sponsor.PrimaryContactID = in.PrimaryContactID
	sponsor.ExpectedClose = in.ExpectedClose
	sponsor.ActualClose = in.ActualClose
	sponsor.LostReason = in.LostReason
	sponsor.Notes = in.Notes
	sponsor.CustomFields = in.CustomFields
	sponsor.Position = in.Position
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

	var input SponsorInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}

	sponsor := &models.Sponsor{OrganizationID: orgID}
	input.applyTo(sponsor)

	if err := h.svc.Create(c.Context(), sponsor); err != nil {
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

	var input SponsorInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}
	input.applyTo(sponsor)

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

type updateStageBody struct {
	Stage    string `json:"stage" validate:"required"`
	Position int    `json:"position"`
}

func (h *SponsorHandler) UpdateStage(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid sponsor id")
	}

	var body updateStageBody
	if verr := reqbind.JSON(c, &body); verr != nil {
		return verr
	}

	if err := h.svc.UpdateStage(c.Context(), orgID, id, body.Stage, body.Position); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update stage")
	}

	return c.JSON(fiber.Map{"message": "stage updated"})
}
