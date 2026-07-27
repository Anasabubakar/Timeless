package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/service"
)

type OnboardingHandler struct {
	svc *service.OnboardingService
}

func NewOnboardingHandler(svc *service.OnboardingService) *OnboardingHandler {
	return &OnboardingHandler{svc: svc}
}

func (h *OnboardingHandler) GetState(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	state, err := h.svc.GetState(c.Context(), orgID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load onboarding state")
	}
	return c.JSON(fiber.Map{"data": state})
}

func (h *OnboardingHandler) SaveState(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	var input service.SaveStateInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	state, err := h.svc.SaveState(c.Context(), orgID, userID, input)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to save onboarding state")
	}
	return c.JSON(fiber.Map{"data": state})
}

func (h *OnboardingHandler) Complete(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	user, err := h.svc.Complete(c.Context(), orgID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to complete onboarding")
	}
	return c.JSON(fiber.Map{"data": user})
}
