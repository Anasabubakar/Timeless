package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(c fiber.Ctx) error {
	var input service.RegisterInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if input.Email == "" || input.Password == "" || input.FirstName == "" || input.LastName == "" || input.OrgName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "all fields are required")
	}

	if len(input.Password) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	user, tokens, err := h.svc.Register(c.Context(), input)
	if err != nil {
		if err.Error() == "email already registered" {
			return fiber.NewError(fiber.StatusConflict, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user":   user,
		"tokens": tokens,
	})
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var input service.LoginInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	user, tokens, err := h.svc.Login(c.Context(), input)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}

	return c.JSON(fiber.Map{
		"user":   user,
		"tokens": tokens,
	})
}

func (h *AuthHandler) RefreshToken(c fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.RefreshToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "refresh_token is required")
	}

	tokens, err := h.svc.RefreshToken(c.Context(), body.RefreshToken)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}

	return c.JSON(fiber.Map{"tokens": tokens})
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.RefreshToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "refresh_token is required")
	}

	_ = h.svc.Logout(c.Context(), body.RefreshToken)
	return c.JSON(fiber.Map{"message": "logged out"})
}

func (h *AuthHandler) Me(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	user, err := h.svc.GetUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	return c.JSON(fiber.Map{"user": user})
}
