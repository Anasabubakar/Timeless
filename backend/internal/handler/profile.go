package handler

import (
	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/repository"
)

type ProfileHandler struct {
	userRepo *repository.UserRepository
}

func NewProfileHandler(userRepo *repository.UserRepository) *ProfileHandler {
	return &ProfileHandler{userRepo: userRepo}
}

func (h *ProfileHandler) Update(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	user, err := h.userRepo.FindByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	var body struct {
		FirstName *string `json:"first_name"`
		LastName  *string `json:"last_name"`
		Phone     *string `json:"phone"`
		JobTitle  *string `json:"job_title"`
		AvatarURL *string `json:"avatar_url"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if body.FirstName != nil {
		user.FirstName = *body.FirstName
	}
	if body.LastName != nil {
		user.LastName = *body.LastName
	}
	if body.Phone != nil {
		user.Phone = body.Phone
	}
	if body.JobTitle != nil {
		user.JobTitle = body.JobTitle
	}
	if body.AvatarURL != nil {
		user.AvatarURL = body.AvatarURL
	}

	if err := h.userRepo.Update(c.Context(), user); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update profile")
	}

	return c.JSON(fiber.Map{"user": user})
}

func (h *ProfileHandler) ChangePassword(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	user, err := h.userRepo.FindByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if body.CurrentPassword == "" || body.NewPassword == "" {
		return fiber.NewError(fiber.StatusBadRequest, "both current and new password are required")
	}

	if len(body.NewPassword) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "new password must be at least 8 characters")
	}

	if user.PasswordHash == nil {
		return fiber.NewError(fiber.StatusBadRequest, "no password set for this account")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(body.CurrentPassword)); err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "current password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to hash password")
	}

	hashStr := string(hash)
	user.PasswordHash = &hashStr
	if err := h.userRepo.Update(c.Context(), user); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update password")
	}

	return c.JSON(fiber.Map{"message": "password updated"})
}
