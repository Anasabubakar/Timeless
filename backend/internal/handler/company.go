package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/pkg/reqbind"
	"github.com/timeless/backend/internal/service"
)

type CompanyHandler struct {
	svc *service.CompanyService
}

func NewCompanyHandler(svc *service.CompanyService) *CompanyHandler {
	return &CompanyHandler{svc: svc}
}

// CompanyInput is the client-writable subset of models.Company. Binding
// requests directly into the GORM model (the previous behavior) let a
// client set ID, OrganizationID, CreatedAt/UpdatedAt/DeletedAt — none of
// which is intentional, but all of which round-trip through Bind().JSON
// with no unknown-field rejection, since they're real (if not meant to
// be client-writable) json tags on Company.
type CompanyInput struct {
	Name           string         `json:"name" validate:"required"`
	Domain         *string        `json:"domain,omitempty"`
	Website        *string        `json:"website,omitempty" validate:"omitempty,url"`
	LogoURL        *string        `json:"logo_url,omitempty" validate:"omitempty,url"`
	Description    *string        `json:"description,omitempty"`
	IndustryID     *uuid.UUID     `json:"industry_id,omitempty"`
	EmployeeCount  *string        `json:"employee_count,omitempty"`
	AnnualRevenue  *string        `json:"annual_revenue,omitempty"`
	Headquarters   *string        `json:"headquarters,omitempty"`
	FoundedYear    *int           `json:"founded_year,omitempty"`
	LinkedinURL    *string        `json:"linkedin_url,omitempty" validate:"omitempty,url"`
	TwitterURL     *string        `json:"twitter_url,omitempty" validate:"omitempty,url"`
	Phone          *string        `json:"phone,omitempty"`
	Address        datatypes.JSON `json:"address,omitempty"`
	Tags           pq.StringArray `json:"tags"`
	EnrichmentData datatypes.JSON `json:"enrichment_data"`
	Score          *int           `json:"score,omitempty"`
	Status         string         `json:"status"`
	Source         *string        `json:"source,omitempty"`
}

// applyTo copies every writable field from in onto company, leaving
// Base/OrganizationID/relations untouched — those are set by the
// handler from trusted context (URL param, auth locals), never from
// the request body.
func (in *CompanyInput) applyTo(company *models.Company) {
	company.Name = in.Name
	company.Domain = in.Domain
	company.Website = in.Website
	company.LogoURL = in.LogoURL
	company.Description = in.Description
	company.IndustryID = in.IndustryID
	company.EmployeeCount = in.EmployeeCount
	company.AnnualRevenue = in.AnnualRevenue
	company.Headquarters = in.Headquarters
	company.FoundedYear = in.FoundedYear
	company.LinkedinURL = in.LinkedinURL
	company.TwitterURL = in.TwitterURL
	company.Phone = in.Phone
	company.Address = in.Address
	company.Tags = in.Tags
	company.EnrichmentData = in.EnrichmentData
	company.Score = in.Score
	if in.Status != "" {
		company.Status = in.Status
	}
	company.Source = in.Source
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

	var input CompanyInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}

	company := &models.Company{OrganizationID: orgID}
	input.applyTo(company)

	if err := h.svc.Create(c.Context(), company); err != nil {
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

	var input CompanyInput
	if verr := reqbind.JSON(c, &input); verr != nil {
		return verr
	}
	input.applyTo(company)

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
