package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/sponsoros/backend/internal/models"
	"github.com/sponsoros/backend/internal/service"
)

type CommunicationHandler struct {
	svc *service.CommunicationService
}

func NewCommunicationHandler(svc *service.CommunicationService) *CommunicationHandler {
	return &CommunicationHandler{svc: svc}
}

func (h *CommunicationHandler) List(c fiber.Ctx) error {
	orgID := uuid.MustParse(c.Locals("org_id").(string))

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	filters := make(map[string]interface{})
	if v := c.Query("status"); v != "" {
		filters["status"] = v
	}
	if v := c.Query("type"); v != "" {
		filters["type"] = v
	}
	if v := c.Query("direction"); v != "" {
		filters["direction"] = v
	}
	if v := c.Query("sponsor_id"); v != "" {
		filters["sponsor_id"] = v
	}
	if v := c.Query("contact_id"); v != "" {
		filters["contact_id"] = v
	}

	comms, total, err := h.svc.List(orgID, filters, limit, offset)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to list communications"})
	}

	return c.JSON(fiber.Map{
		"data":  comms,
		"total": total,
	})
}

func (h *CommunicationHandler) Get(c fiber.Ctx) error {
	orgID := uuid.MustParse(c.Locals("org_id").(string))
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	comm, err := h.svc.GetByID(orgID, id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "communication not found"})
	}

	return c.JSON(fiber.Map{"data": comm})
}

func (h *CommunicationHandler) Create(c fiber.Ctx) error {
	orgID := uuid.MustParse(c.Locals("org_id").(string))
	userID := uuid.MustParse(c.Locals("user_id").(string))

	var comm models.Communication
	if err := c.Bind().JSON(&comm); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	comm.OrganizationID = orgID
	comm.SentBy = &userID

	if err := h.svc.Create(&comm); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create communication"})
	}

	return c.Status(201).JSON(fiber.Map{"data": comm})
}

func (h *CommunicationHandler) Update(c fiber.Ctx) error {
	orgID := uuid.MustParse(c.Locals("org_id").(string))
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	existing, err := h.svc.GetByID(orgID, id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "communication not found"})
	}

	if err := c.Bind().JSON(existing); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := h.svc.Update(existing); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update communication"})
	}

	return c.JSON(fiber.Map{"data": existing})
}

func (h *CommunicationHandler) Delete(c fiber.Ctx) error {
	orgID := uuid.MustParse(c.Locals("org_id").(string))
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.svc.Delete(orgID, id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete communication"})
	}

	return c.JSON(fiber.Map{"message": "deleted"})
}

func (h *CommunicationHandler) Stats(c fiber.Ctx) error {
	orgID := uuid.MustParse(c.Locals("org_id").(string))

	stats, err := h.svc.GetStats(orgID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to get stats"})
	}

	return c.JSON(fiber.Map{"data": stats})
}
