package handler

import (
	"encoding/json"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

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

// Update handles two distinct kinds of change with different
// authorization requirements:
//   - LogoURL/Domain/Settings: any caller with settings:write (route-
//     level RBAC already enforces this).
//   - Name/Slug/Password: identity-changing fields the organization
//     security requirements gate behind Owner + current_password
//     reverification, handled by OrganizationService.UpdateSecure.
//
// A request can touch both kinds at once; the secure fields are applied
// first (and fail the whole request if unauthorized) so a caller can't
// smuggle a name change through by burying it among fields they are
// allowed to touch.
func (h *OrganizationHandler) Update(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)
	ip := c.IP()

	var body struct {
		Name            *string                `json:"name"`
		Slug            *string                `json:"slug"`
		Password        *string                `json:"password"`
		CurrentPassword string                 `json:"current_password"`
		LogoURL         *string                `json:"logo_url"`
		Domain          *string                `json:"domain"`
		Settings        map[string]interface{} `json:"settings"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	touchesIdentity := body.Name != nil || body.Slug != nil || body.Password != nil

	org, err := h.svc.GetByID(c.Context(), orgID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "organization not found")
	}

	if touchesIdentity {
		if body.CurrentPassword == "" {
			return fiber.NewError(fiber.StatusBadRequest, "current_password is required to change name, slug, or password")
		}
		updated, err := h.svc.UpdateSecure(c.Context(), orgID, userID, service.UpdateSecureInput{
			Name:        body.Name,
			Slug:        body.Slug,
			NewPassword: body.Password,
		}, body.CurrentPassword, ip)
		if err != nil {
			return mapOrgSecureError(err)
		}
		org = updated
	}

	if body.LogoURL != nil || body.Domain != nil || body.Settings != nil {
		if body.LogoURL != nil {
			org.LogoURL = body.LogoURL
		}
		if body.Domain != nil {
			org.Domain = body.Domain
		}
		if body.Settings != nil {
			settingsJSON, err := json.Marshal(body.Settings)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid settings")
			}
			org.Settings = settingsJSON
		}
		if err := h.svc.Update(c.Context(), org); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update organization")
		}
	}

	return c.JSON(fiber.Map{"organization": org})
}

type TransferOwnershipInput struct {
	NewOwnerID      string `json:"new_owner_id" validate:"required"`
	CurrentPassword string `json:"current_password" validate:"required"`
}

// TransferOwnership: POST /organizations/current/transfer-ownership —
// Owner-only, requires the current organization password. Moves Owner
// to another member and demotes the caller to Admin.
func (h *OrganizationHandler) TransferOwnership(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	var input TransferOwnershipInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	targetID, err := uuid.Parse(input.NewOwnerID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid new_owner_id")
	}

	if err := h.svc.TransferOwnership(c.Context(), orgID, userID, targetID, input.CurrentPassword, c.IP()); err != nil {
		return mapOrgSecureError(err)
	}

	return c.JSON(fiber.Map{"message": "ownership transferred"})
}

func mapOrgSecureError(err error) error {
	switch {
	case errors.Is(err, service.ErrNotOwner):
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrOrgPasswordLocked):
		return fiber.NewError(fiber.StatusTooManyRequests, err.Error())
	case errors.Is(err, service.ErrIncorrectOrgPassword):
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	case errors.Is(err, service.ErrSlugTaken):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}
	return fiber.NewError(fiber.StatusInternalServerError, "failed to update organization")
}
