package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/sponsoros/backend/internal/middleware"
	"github.com/sponsoros/backend/internal/models"
	"github.com/sponsoros/backend/internal/service"
)

type WebhookHandler struct {
	svc *service.WebhookService
}

func NewWebhookHandler(svc *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

func (h *WebhookHandler) List(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	webhooks, err := h.svc.List(c.Context(), orgID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list webhooks")
	}
	return c.JSON(fiber.Map{"data": webhooks})
}

func (h *WebhookHandler) Get(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	webhook, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "webhook not found")
	}
	return c.JSON(fiber.Map{"data": webhook})
}

func (h *WebhookHandler) Create(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var body struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.URL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "url is required")
	}

	webhook := &models.Webhook{
		OrganizationID: orgID,
		URL:            body.URL,
		IsActive:       true,
	}

	if len(body.Events) > 0 {
		eventsJSON, _ := json.Marshal(body.Events)
		webhook.Events = eventsJSON
	}

	if err := h.svc.Create(c.Context(), webhook); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create webhook")
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": webhook})
}

func (h *WebhookHandler) Update(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	webhook, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "webhook not found")
	}

	var body struct {
		URL      *string  `json:"url"`
		Events   []string `json:"events"`
		IsActive *bool    `json:"is_active"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if body.URL != nil {
		webhook.URL = *body.URL
	}
	if body.Events != nil {
		eventsJSON, _ := json.Marshal(body.Events)
		webhook.Events = eventsJSON
	}
	if body.IsActive != nil {
		webhook.IsActive = *body.IsActive
		if *body.IsActive {
			webhook.FailureCount = 0
		}
	}

	if err := h.svc.Update(c.Context(), webhook); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update webhook")
	}
	return c.JSON(fiber.Map{"data": webhook})
}

func (h *WebhookHandler) Delete(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := h.svc.Delete(c.Context(), orgID, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete webhook")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *WebhookHandler) RotateSecret(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	webhook, err := h.svc.RotateSecret(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to rotate secret")
	}
	return c.JSON(fiber.Map{"data": webhook, "secret": webhook.Secret})
}

func (h *WebhookHandler) Test(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := h.svc.TestWebhook(c.Context(), orgID, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to send test")
	}
	return c.JSON(fiber.Map{"message": "test webhook enqueued"})
}

func (h *WebhookHandler) Deliveries(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	deliveries, err := h.svc.ListDeliveries(c.Context(), orgID, id, 50)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list deliveries")
	}
	return c.JSON(fiber.Map{"data": deliveries})
}
