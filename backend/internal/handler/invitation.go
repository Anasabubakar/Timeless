package handler

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v3"

	"github.com/timeless/backend/internal/pkg/reqbind"
	"github.com/timeless/backend/internal/service"
)

type InvitationHandler struct {
	svc *service.InvitationService
}

func NewInvitationHandler(svc *service.InvitationService) *InvitationHandler {
	return &InvitationHandler{svc: svc}
}

// Accept: POST /invitations/accept — public. The invited person proves
// they hold the mailed token and sets a password, which creates their
// account for the first time and logs them straight in.
func (h *InvitationHandler) Accept(c fiber.Ctx) error {
	var input service.AcceptInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}

	user, tokens, err := h.svc.Accept(c.Context(), input, sessionMeta(c, false))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvitationInvalid):
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		case err.Error() == "email already registered":
			return fiber.NewError(fiber.StatusConflict, err.Error())
		}
		log.Printf("invitation: accept failed: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to accept invitation")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user":   user,
		"tokens": tokens,
	})
}
