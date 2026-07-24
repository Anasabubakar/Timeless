package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/service"
)

type ActivityHandler struct {
	svc *service.ActivityService
}

func NewActivityHandler(svc *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{svc: svc}
}

func (h *ActivityHandler) List(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	entityType := c.Query("entity_type")
	entityID := c.Query("entity_id")
	action := c.Query("type")

	activities, total, err := h.svc.List(c.Context(), orgID, limit, offset, entityType, entityID, action)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list activities")
	}

	return c.JSON(fiber.Map{
		"data":   activities,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *ActivityHandler) Create(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	var activity models.Activity
	if err := c.Bind().JSON(&activity); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	activity.OrganizationID = orgID
	activity.UserID = &userID
	if err := h.svc.Create(c.Context(), &activity); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create activity")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": activity})
}
