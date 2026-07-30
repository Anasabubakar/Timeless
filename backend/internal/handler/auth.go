package handler

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/pkg/reqbind"
	"github.com/timeless/backend/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// sessionMeta captures device/request info from the incoming HTTP
// request so the service layer can persist a meaningful session record
// without importing fiber.
func sessionMeta(c fiber.Ctx, rememberMe bool) service.SessionMeta {
	return service.SessionMeta{
		IP:         c.IP(),
		UserAgent:  c.Get("User-Agent"),
		RememberMe: rememberMe,
	}
}

func (h *AuthHandler) Register(c fiber.Ctx) error {
	var input service.RegisterInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}

	user, tokens, err := h.svc.Register(c.Context(), input, sessionMeta(c, false))
	if err != nil {
		if err.Error() == "email already registered" {
			return fiber.NewError(fiber.StatusConflict, err.Error())
		}
		log.Printf("auth: registration failed for %s: %v", input.Email, err)
		return fiber.NewError(fiber.StatusInternalServerError, "registration failed")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user":   user,
		"tokens": tokens,
	})
}

// LookupOrganization: GET /auth/organizations/lookup?name=... — public,
// used by the signup form to decide whether to render "create a new
// organization" (prompting for a new org password) or "join this
// organization" (prompting for its existing password) before any account
// is created.
func (h *AuthHandler) LookupOrganization(c fiber.Ctx) error {
	name := c.Query("name")
	if name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	result, err := h.svc.LookupOrganization(c.Context(), name)
	if err != nil {
		log.Printf("auth: organization lookup failed for %q: %v", name, err)
		return fiber.NewError(fiber.StatusInternalServerError, "lookup failed")
	}

	return c.JSON(result)
}

// Join: POST /auth/join — CASE 2 of signup, joining an organization that
// already exists by proving knowledge of its shared password.
func (h *AuthHandler) Join(c fiber.Ctx) error {
	var input service.JoinInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}

	user, tokens, err := h.svc.JoinOrganization(c.Context(), input, sessionMeta(c, false))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrgPasswordLocked):
			return fiber.NewError(fiber.StatusTooManyRequests, err.Error())
		case errors.Is(err, service.ErrIncorrectOrgPassword):
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		case err.Error() == "email already registered":
			return fiber.NewError(fiber.StatusConflict, err.Error())
		case err.Error() == "organization not found":
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		log.Printf("auth: join failed for %s: %v", input.Email, err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to join organization")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user":   user,
		"tokens": tokens,
	})
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var input service.LoginInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}

	user, tokens, err := h.svc.Login(c.Context(), input, sessionMeta(c, input.RememberMe))
	if err != nil {
		if errors.Is(err, service.ErrAccountLocked) {
			return fiber.NewError(fiber.StatusTooManyRequests, err.Error())
		}
		if errors.Is(err, service.ErrMFARequired) {
			ticket, ticketErr := h.svc.IssueMFAPendingTicket(user)
			if ticketErr != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "failed to start mfa verification")
			}
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"mfa_required": true,
				"mfa_ticket":   ticket,
			})
		}
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}

	return c.JSON(fiber.Map{
		"user":   user,
		"tokens": tokens,
	})
}

type refreshTokenBody struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (h *AuthHandler) RefreshToken(c fiber.Ctx) error {
	var body refreshTokenBody
	if verr := reqbind.JSON(c, &body); verr != nil {
		return verr
	}

	tokens, err := h.svc.RefreshToken(c.Context(), body.RefreshToken)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}

	return c.JSON(fiber.Map{"tokens": tokens})
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	var body refreshTokenBody
	if verr := reqbind.JSON(c, &body); verr != nil {
		return verr
	}

	_ = h.svc.Logout(c.Context(), body.RefreshToken, middleware.GetOrgID(c), middleware.GetUserID(c), c.IP())
	return c.JSON(fiber.Map{"message": "logged out"})
}

type emailOnlyBody struct {
	Email string `json:"email" validate:"required,email"`
}

func (h *AuthHandler) ForgotPassword(c fiber.Ctx) error {
	var body emailOnlyBody
	if verr := reqbind.JSON(c, &body); verr != nil {
		return verr
	}

	_ = h.svc.ForgotPassword(c.Context(), body.Email, c.IP())
	return c.JSON(fiber.Map{"message": "if that address has an account, a reset email has been sent"})
}

type resetPasswordBody struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

func (h *AuthHandler) ResetPassword(c fiber.Ctx) error {
	var body resetPasswordBody
	if verr := reqbind.JSON(c, &body); verr != nil {
		return verr
	}

	if err := h.svc.ResetPassword(c.Context(), body.Token, body.NewPassword, c.IP()); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{"message": "password reset — all sessions have been signed out"})
}

type tokenOnlyBody struct {
	Token string `json:"token" validate:"required"`
}

func (h *AuthHandler) VerifyEmail(c fiber.Ctx) error {
	var body tokenOnlyBody
	if verr := reqbind.JSON(c, &body); verr != nil {
		return verr
	}

	if err := h.svc.VerifyEmail(c.Context(), body.Token); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{"message": "email verified"})
}

func (h *AuthHandler) ResendVerification(c fiber.Ctx) error {
	var body emailOnlyBody
	if verr := reqbind.JSON(c, &body); verr != nil {
		return verr
	}

	_ = h.svc.ResendVerification(c.Context(), body.Email)
	// Always a generic success response — do not reveal whether the
	// address exists or is already verified.
	return c.JSON(fiber.Map{"message": "if that address needs verification, an email has been sent"})
}

type verifyMFALoginBody struct {
	Ticket     string `json:"mfa_ticket" validate:"required"`
	Code       string `json:"code" validate:"required,len=6|min=10"`
	RememberMe bool   `json:"remember_me"`
}

func (h *AuthHandler) VerifyMFALogin(c fiber.Ctx) error {
	var body verifyMFALoginBody
	if verr := reqbind.JSON(c, &body); verr != nil {
		return verr
	}

	user, tokens, err := h.svc.VerifyMFALogin(c.Context(), body.Ticket, body.Code, sessionMeta(c, body.RememberMe))
	if err != nil {
		if errors.Is(err, service.ErrAccountLocked) {
			return fiber.NewError(fiber.StatusTooManyRequests, err.Error())
		}
		return fiber.NewError(fiber.StatusUnauthorized, "invalid verification code")
	}

	return c.JSON(fiber.Map{"user": user, "tokens": tokens})
}

func (h *AuthHandler) EnrollMFA(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	enrollment, err := h.svc.EnrollMFA(c.Context(), userID)
	if err != nil {
		log.Printf("auth: mfa enrollment failed for user %s: %v", userID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to start mfa enrollment")
	}
	return c.JSON(enrollment)
}

type mfaCodeBody struct {
	Code string `json:"code" validate:"required,len=6|min=10"`
}

func (h *AuthHandler) ConfirmMFA(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	var body mfaCodeBody
	if verr := reqbind.JSON(c, &body); verr != nil {
		return verr
	}

	if err := h.svc.ConfirmMFA(c.Context(), userID, body.Code, c.IP()); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"message": "mfa enabled"})
}

type passwordOnlyBody struct {
	Password string `json:"password" validate:"required"`
}

func (h *AuthHandler) DisableMFA(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	var body passwordOnlyBody
	if verr := reqbind.JSON(c, &body); verr != nil {
		return verr
	}

	if err := h.svc.DisableMFA(c.Context(), userID, body.Password, c.IP()); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"message": "mfa disabled"})
}

func (h *AuthHandler) ListSessions(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	sessions, err := h.svc.ListSessions(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list sessions")
	}
	return c.JSON(fiber.Map{"sessions": sessions})
}

func (h *AuthHandler) RevokeSession(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid session id")
	}

	if err := h.svc.RevokeSession(c.Context(), userID, sessionID, middleware.GetOrgID(c), c.IP()); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	return c.JSON(fiber.Map{"message": "session revoked"})
}

func (h *AuthHandler) LogoutAllSessions(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	if err := h.svc.LogoutAllSessions(c.Context(), userID, middleware.GetOrgID(c), c.IP()); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to revoke sessions")
	}
	return c.JSON(fiber.Map{"message": "logged out of all sessions"})
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
