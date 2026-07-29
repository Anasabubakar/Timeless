package handler

import (
	"github.com/timeless/backend/internal/logging"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/service"
)

type ProposalHandler struct {
	svc *service.ProposalService
}

func NewProposalHandler(svc *service.ProposalService) *ProposalHandler {
	return &ProposalHandler{svc: svc}
}

func (h *ProposalHandler) List(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	status := c.Query("status")

	var sponsorID *uuid.UUID
	if sid := c.Query("sponsor_id"); sid != "" {
		parsed, err := uuid.Parse(sid)
		if err == nil {
			sponsorID = &parsed
		}
	}

	proposals, total, err := h.svc.List(c.Context(), orgID, sponsorID, status)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list proposals")
	}

	return c.JSON(fiber.Map{"data": proposals, "total": total})
}

func (h *ProposalHandler) Get(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	proposal, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "proposal not found")
	}

	return c.JSON(fiber.Map{"data": proposal})
}

func (h *ProposalHandler) Create(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	var proposal models.Proposal
	if err := c.Bind().JSON(&proposal); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	proposal.OrganizationID = orgID
	proposal.CreatedBy = &userID
	if err := h.svc.Create(c.Context(), &proposal); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create proposal")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": proposal})
}

func (h *ProposalHandler) Update(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	proposal, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "proposal not found")
	}

	if err := c.Bind().JSON(proposal); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// Same re-pin as CompanyHandler/ContactHandler.Update: Update() saves
	// by primary key with no org check, so a client-supplied
	// "organization_id" in the body would otherwise move this proposal
	// into a different tenant's org.
	proposal.OrganizationID = orgID
	proposal.ID = id
	if err := h.svc.Update(c.Context(), proposal); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update proposal")
	}

	return c.JSON(fiber.Map{"data": proposal})
}

func (h *ProposalHandler) Delete(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := h.svc.Delete(c.Context(), orgID, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete proposal")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ProposalHandler) Generate(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	var input service.GenerateProposalInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	proposal, err := h.svc.Generate(c.Context(), orgID, userID, &input)
	if err != nil {
		logging.Printf("proposal: generation failed for org %s: %v", orgID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "proposal generation failed")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": proposal})
}
