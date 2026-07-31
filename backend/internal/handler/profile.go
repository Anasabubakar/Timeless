package handler

import (
	"errors"

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

type deleteAccountBody struct {
	Password           string `json:"password" validate:"required"`
	ConfirmOrgDeletion bool   `json:"confirm_org_deletion"`
}

// DeleteAccount: POST /profile/delete — self-service account deletion.
// A DELETE verb would be more conventional, but this needs a JSON body
// (password + confirmation), and DELETE-with-body support is
// inconsistent enough across clients/proxies that every other
// password-gated action in this API (ChangePassword above,
// OrganizationService.UpdateSecure) already uses POST for the same
// reason.
func (h *ProfileHandler) DeleteAccount(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var body deleteAccountBody
	if verr := reqbind.JSON(c, &body); verr != nil {
		return verr
	}

	err := h.authSvc.DeleteAccount(c.Context(), userID, service.DeleteAccountInput{
		Password:           body.Password,
		ConfirmOrgDeletion: body.ConfirmOrgDeletion,
	}, c.IP())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMustTransferOwnershipFirst), errors.Is(err, service.ErrMustConfirmOrgDeletion):
			return fiber.NewError(fiber.StatusConflict, err.Error())
		case err.Error() == "password is incorrect":
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete account")
	}

	return c.JSON(fiber.Map{"message": "your account has been deleted"})
}
