package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/sponsoros/backend/internal/middleware"
	"github.com/sponsoros/backend/internal/models"
	"github.com/sponsoros/backend/internal/service"
)

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

type NotificationHandler struct {
	svc *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) List(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	unreadOnly := c.Query("unread") == "true"

	if limit > 100 {
		limit = 100
	}

	notifications, total, err := h.svc.List(c.Context(), orgID, userID, unreadOnly, limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to list notifications")
	}

	return c.JSON(fiber.Map{
		"data":  notifications,
		"total": total,
	})
}

func (h *NotificationHandler) UnreadCount(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	count, err := h.svc.UnreadCount(c.Context(), orgID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to count notifications")
	}

	return c.JSON(fiber.Map{"count": count})
}

func (h *NotificationHandler) MarkRead(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)
	notifID, err := parseUUID(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid notification ID")
	}

	if err := h.svc.MarkRead(c.Context(), orgID, userID, notifID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to mark notification read")
	}

	return c.JSON(fiber.Map{"message": "Marked as read"})
}

func (h *NotificationHandler) MarkAllRead(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	if err := h.svc.MarkAllRead(c.Context(), orgID, userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to mark all as read")
	}

	return c.JSON(fiber.Map{"message": "All marked as read"})
}

func (h *NotificationHandler) Delete(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)
	notifID, err := parseUUID(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid notification ID")
	}

	if err := h.svc.Delete(c.Context(), orgID, userID, notifID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete notification")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *NotificationHandler) GetPreferences(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	prefs, err := h.svc.GetPreferences(c.Context(), orgID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get preferences")
	}

	return c.JSON(fiber.Map{"data": prefs})
}

type UpdatePreferenceRequest struct {
	Type  string `json:"type" validate:"required"`
	InApp *bool  `json:"in_app"`
	Email *bool  `json:"email"`
}

func (h *NotificationHandler) UpdatePreference(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	var req UpdatePreferenceRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	pref := &models.NotificationPreference{
		OrganizationID: orgID,
		UserID:         userID,
		Type:           models.NotificationType(req.Type),
		InApp:          true,
		Email:          false,
	}

	if req.InApp != nil {
		pref.InApp = *req.InApp
	}
	if req.Email != nil {
		pref.Email = *req.Email
	}

	if err := h.svc.UpdatePreference(c.Context(), pref); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update preference")
	}

	return c.JSON(pref)
}
