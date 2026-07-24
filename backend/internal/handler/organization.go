package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/service"
)

type OrganizationHandler struct {
	svc *service.OrganizationService
}

func NewOrganizationHandler(svc *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{svc: svc}
}

func (h *OrganizationHandler) GetCurrent(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	org, err := h.svc.GetByID(c.Context(), orgID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "organization not found")
	}
	return c.JSON(fiber.Map{"organization": org})
}

func (h *OrganizationHandler) Update(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	org, err := h.svc.GetByID(c.Context(), orgID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "organization not found")
	}

	var body struct {
		Name     *string                `json:"name"`
		Slug     *string                `json:"slug"`
		LogoURL  *string                `json:"logo_url"`
		Domain   *string                `json:"domain"`
		Settings map[string]interface{} `json:"settings"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if body.Name != nil {
		org.Name = *body.Name
	}
	if body.Slug != nil {
		org.Slug = *body.Slug
	}
	if body.LogoURL != nil {
		org.LogoURL = body.LogoURL
	}
	if body.Domain != nil {
		org.Domain = body.Domain
	}
	if body.Settings != nil {
		settingsJSON, _ := json.Marshal(body.Settings)
		org.Settings = settingsJSON
	}

	if err := h.svc.Update(c.Context(), org); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update organization")
	}

	return c.JSON(fiber.Map{"organization": org})
}
