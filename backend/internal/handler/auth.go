package handler

import (
	"errors"
	"log"

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
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if input.Email == "" || input.Password == "" || input.FirstName == "" || input.LastName == "" || input.OrgName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "all fields are required")
	}

	if len(input.Password) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
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

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var input service.LoginInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
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

func (h *AuthHandler) ForgotPassword(c fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email is required")
	}

	_ = h.svc.ForgotPassword(c.Context(), body.Email, c.IP())
	return c.JSON(fiber.Map{"message": "if that address has an account, a reset email has been sent"})
}

func (h *AuthHandler) ResetPassword(c fiber.Ctx) error {
	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Token == "" || body.NewPassword == "" {
		return fiber.NewError(fiber.StatusBadRequest, "token and new_password are required")
	}

	if err := h.svc.ResetPassword(c.Context(), body.Token, body.NewPassword); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{"message": "password reset — all sessions have been signed out"})
}

func (h *AuthHandler) VerifyEmail(c fiber.Ctx) error {
	var body struct {
		Token string `json:"token"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Token == "" {
		return fiber.NewError(fiber.StatusBadRequest, "token is required")
	}

	if err := h.svc.VerifyEmail(c.Context(), body.Token); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{"message": "email verified"})
}

func (h *AuthHandler) ResendVerification(c fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email is required")
	}

	_ = h.svc.ResendVerification(c.Context(), body.Email)
	// Always a generic success response — do not reveal whether the
	// address exists or is already verified.
	return c.JSON(fiber.Map{"message": "if that address needs verification, an email has been sent"})
}

func (h *AuthHandler) VerifyMFALogin(c fiber.Ctx) error {
	var body struct {
		Ticket     string `json:"mfa_ticket"`
		Code       string `json:"code"`
		RememberMe bool   `json:"remember_me"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Ticket == "" || body.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "mfa_ticket and code are required")
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

func (h *AuthHandler) ConfirmMFA(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	var body struct {
		Code string `json:"code"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "code is required")
	}

	if err := h.svc.ConfirmMFA(c.Context(), userID, body.Code); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"message": "mfa enabled"})
}

func (h *AuthHandler) DisableMFA(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "password is required")
	}

	if err := h.svc.DisableMFA(c.Context(), userID, body.Password); err != nil {
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

	if err := h.svc.RevokeSession(c.Context(), userID, sessionID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	return c.JSON(fiber.Map{"message": "session revoked"})
}

func (h *AuthHandler) LogoutAllSessions(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	if err := h.svc.LogoutAllSessions(c.Context(), userID); err != nil {
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
