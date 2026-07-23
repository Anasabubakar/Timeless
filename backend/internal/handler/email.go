package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/sponsoros/backend/internal/email"
	"github.com/sponsoros/backend/internal/middleware"
	"github.com/sponsoros/backend/internal/worker"
)

type EmailHandler struct {
	sender       *email.Sender
	workerClient *worker.Client
}

func NewEmailHandler(sender *email.Sender, workerClient *worker.Client) *EmailHandler {
	return &EmailHandler{sender: sender, workerClient: workerClient}
}

type SendEmailRequest struct {
	To       []string `json:"to" validate:"required,min=1"`
	CC       []string `json:"cc"`
	BCC      []string `json:"bcc"`
	Subject  string   `json:"subject" validate:"required"`
	TextBody string   `json:"text_body"`
	HTMLBody string   `json:"html_body"`
	ReplyTo  string   `json:"reply_to"`
}

type SendTemplateEmailRequest struct {
	To           []string          `json:"to" validate:"required,min=1"`
	CC           []string          `json:"cc"`
	BCC          []string          `json:"bcc"`
	Subject      string            `json:"subject" validate:"required"`
	TemplateBody string            `json:"template_body" validate:"required"`
	Variables    map[string]string `json:"variables"`
	ReplyTo      string            `json:"reply_to"`
}

func (h *EmailHandler) Send(c fiber.Ctx) error {
	var req SendEmailRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if len(req.To) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "At least one recipient required")
	}

	if req.TextBody == "" && req.HTMLBody == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Either text_body or html_body required")
	}

	orgID := middleware.GetOrgID(c)

	payload := worker.EmailPayload{
		OrgID:    orgID.String(),
		To:       req.To,
		CC:       req.CC,
		BCC:      req.BCC,
		Subject:  req.Subject,
		TextBody: req.TextBody,
		HTMLBody: req.HTMLBody,
		ReplyTo:  req.ReplyTo,
	}

	taskID, err := h.workerClient.EnqueueEmail(payload)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to queue email")
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message": "Email queued for delivery",
		"task_id": taskID,
	})
}

func (h *EmailHandler) SendDirect(c fiber.Ctx) error {
	var req SendEmailRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if len(req.To) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "At least one recipient required")
	}

	msg := &email.Message{
		To:       req.To,
		CC:       req.CC,
		BCC:      req.BCC,
		Subject:  req.Subject,
		TextBody: req.TextBody,
		HTMLBody: req.HTMLBody,
		ReplyTo:  req.ReplyTo,
		Tags:     map[string]string{"org_id": middleware.GetOrgID(c).String()},
	}

	result, err := h.sender.Send(c.Context(), msg)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to send email: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"message_id": result.MessageID,
		"provider":   result.ProviderID,
		"status":     result.Status,
	})
}

func (h *EmailHandler) SendTemplate(c fiber.Ctx) error {
	var req SendTemplateEmailRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if len(req.To) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "At least one recipient required")
	}

	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)
	_ = userID

	msg := &email.Message{
		To:      req.To,
		CC:      req.CC,
		BCC:     req.BCC,
		Subject: req.Subject,
		ReplyTo: req.ReplyTo,
		Tags:    map[string]string{"org_id": orgID.String()},
	}

	data := email.TemplateData{
		Subject:   req.Subject,
		Variables: req.Variables,
	}

	result, err := h.sender.SendTemplate(c.Context(), msg, req.TemplateBody, data)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to send email: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"message_id": result.MessageID,
		"provider":   result.ProviderID,
		"status":     result.Status,
	})
}
