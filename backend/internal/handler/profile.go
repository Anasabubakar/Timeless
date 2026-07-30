package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/pkg/reqbind"
	"github.com/timeless/backend/internal/repository"
	"github.com/timeless/backend/internal/service"
)

type ProfileHandler struct {
	userRepo *repository.UserRepository
	authSvc  *service.AuthService
}

func NewProfileHandler(userRepo *repository.UserRepository, authSvc *service.AuthService) *ProfileHandler {
	return &ProfileHandler{userRepo: userRepo, authSvc: authSvc}
}

type updateProfileBody struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	JobTitle  *string `json:"job_title,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty" validate:"omitempty,url"`
}

func (h *ProfileHandler) Update(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	user, err := h.userRepo.FindByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	var body updateProfileBody
	if verr := reqbind.JSON(c, &body); verr != nil {
		return verr
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

type changePasswordBody struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

// ChangePassword delegates to AuthService.ChangePassword rather than
// updating PasswordHash directly (the previous behavior) — that method
// also revokes every other session and sends a notification email,
// exactly like ResetPassword already does. A "logged in and knows the
// current password" password change was previously the one path that
// left any stolen session alive after the password it was stolen
// alongside got rotated.
func (h *ProfileHandler) ChangePassword(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var body changePasswordBody
	if verr := reqbind.JSON(c, &body); verr != nil {
		return verr
	}

	if err := h.authSvc.ChangePassword(c.Context(), userID, body.CurrentPassword, body.NewPassword, c.IP()); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{"message": "password updated — all other sessions have been signed out"})
}
