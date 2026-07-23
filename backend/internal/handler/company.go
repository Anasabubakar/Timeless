package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/sponsoros/backend/internal/middleware"
	"github.com/sponsoros/backend/internal/models"
	"github.com/sponsoros/backend/internal/service"
)

type CompanyHandler struct {
	svc *service.CompanyService
}

func NewCompanyHandler(svc *service.CompanyService) *CompanyHandler {
	return &CompanyHandler{svc: svc}
}

func (h *CompanyHandler) List(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	search := c.Query("search")

	companies, total, err := h.svc.List(c.Context(), orgID, limit, offset, search)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list companies")
	}

	return c.JSON(fiber.Map{
		"companies": companies,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

func (h *CompanyHandler) Create(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var company models.Company
	if err := c.Bind().JSON(&company); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	company.OrganizationID = orgID
	if err := h.svc.Create(c.Context(), &company); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create company")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"company": company})
}

func (h *CompanyHandler) Get(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid company id")
	}

	company, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "company not found")
	}

	return c.JSON(fiber.Map{"company": company})
}

func (h *CompanyHandler) Update(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid company id")
	}

	company, err := h.svc.GetByID(c.Context(), orgID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "company not found")
	}

	if err := c.Bind().JSON(company); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	company.OrganizationID = orgID
	company.ID = id
	if err := h.svc.Update(c.Context(), company); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update company")
	}

	return c.JSON(fiber.Map{"company": company})
}

func (h *CompanyHandler) Delete(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid company id")
	}

	if err := h.svc.Delete(c.Context(), orgID, id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete company")
	}

	return c.JSON(fiber.Map{"message": "company deleted"})
}
