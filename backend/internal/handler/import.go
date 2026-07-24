package handler

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sponsoros/backend/internal/middleware"
	"github.com/sponsoros/backend/internal/models"
)

type ImportHandler struct {
	db *gorm.DB
}

func NewImportHandler(db *gorm.DB) *ImportHandler {
	return &ImportHandler{db: db}
}

func (h *ImportHandler) ImportSponsors(c fiber.Ctx) error  { return h.importEntity(c, "sponsors") }
func (h *ImportHandler) ImportContacts(c fiber.Ctx) error  { return h.importEntity(c, "contacts") }
func (h *ImportHandler) ImportCompanies(c fiber.Ctx) error { return h.importEntity(c, "companies") }

func (h *ImportHandler) importEntity(c fiber.Ctx, entity string) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "file upload required")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "cannot open file")
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil && err != io.EOF {
		return fiber.NewError(fiber.StatusBadRequest, "invalid csv: "+err.Error())
	}
	if len(records) < 2 {
		return fiber.NewError(fiber.StatusBadRequest, "csv must have a header row and at least one data row")
	}

	orgID := middleware.GetOrgID(c)
	headers := normalizeHeaders(records[0])

	var inserted, errCount int
	var rowErrors []map[string]interface{}

	for i, row := range records[1:] {
		if len(row) == 0 || (len(row) == 1 && row[0] == "") {
			continue
		}
		rowMap := mapRow(headers, row)
		var insertErr error
		switch entity {
		case "companies":
			insertErr = h.insertCompany(orgID, rowMap)
		case "contacts":
			insertErr = h.insertContact(orgID, rowMap)
		case "sponsors":
			insertErr = h.insertSponsor(orgID, rowMap)
		}
		if insertErr != nil {
			errCount++
			rowErrors = append(rowErrors, map[string]interface{}{"row": i + 2, "error": insertErr.Error()})
		} else {
			inserted++
		}
	}

	return c.JSON(fiber.Map{
		"entity":     entity,
		"inserted":   inserted,
		"errors":     errCount,
		"row_errors": rowErrors,
	})
}

func (h *ImportHandler) insertCompany(orgID uuid.UUID, row map[string]string) error {
	name := row["name"]
	if name == "" {
		return fmt.Errorf("name is required")
	}
	company := models.Company{
		OrganizationID: orgID,
		Name:           name,
		Status:         "active",
	}
	if v := row["domain"]; v != "" {
		company.Domain = &v
	}
	if v := row["website"]; v != "" {
		company.Website = &v
	}
	if v := row["description"]; v != "" {
		company.Description = &v
	}
	if v := row["employee_count"]; v != "" {
		company.EmployeeCount = &v
	}
	if v := row["annual_revenue"]; v != "" {
		company.AnnualRevenue = &v
	}
	if v := row["headquarters"]; v != "" {
		company.Headquarters = &v
	}
	if v := row["phone"]; v != "" {
		company.Phone = &v
	}
	if v := row["linkedin_url"]; v != "" {
		company.LinkedinURL = &v
	}
	if v := row["twitter_url"]; v != "" {
		company.TwitterURL = &v
	}
	if v := row["source"]; v != "" {
		company.Source = &v
	}
	if v := row["status"]; v != "" {
		company.Status = v
	}
	if v := row["founded_year"]; v != "" {
		if y, err := strconv.Atoi(v); err == nil {
			company.FoundedYear = &y
		}
	}
	return h.db.Create(&company).Error
}

func (h *ImportHandler) insertContact(orgID uuid.UUID, row map[string]string) error {
	firstName, lastName := row["first_name"], row["last_name"]
	if firstName == "" {
		if full := row["name"]; full != "" {
			parts := strings.SplitN(full, " ", 2)
			firstName = parts[0]
			if len(parts) > 1 {
				lastName = parts[1]
			}
		}
	}
	if firstName == "" {
		return fmt.Errorf("first_name (or name) is required")
	}
	contact := models.Contact{
		OrganizationID: orgID,
		FirstName:      firstName,
		LastName:       lastName,
		Status:         "active",
	}
	if v := row["email"]; v != "" {
		contact.Email = &v
	}
	if v := row["phone"]; v != "" {
		contact.Phone = &v
	}
	if v := row["title"]; v != "" {
		contact.Title = &v
	}
	if v := row["department"]; v != "" {
		contact.Department = &v
	}
	if v := row["linkedin_url"]; v != "" {
		contact.LinkedinURL = &v
	}
	if v := row["notes"]; v != "" {
		contact.Notes = &v
	}
	if v := row["status"]; v != "" {
		contact.Status = v
	}
	if v := row["company_id"]; v != "" {
		if id, err := uuid.Parse(v); err == nil {
			contact.CompanyID = &id
		}
	} else if v := row["company_name"]; v != "" {
		var company models.Company
		if err := h.db.Where("organization_id = ? AND name = ?", orgID, v).First(&company).Error; err == nil {
			contact.CompanyID = &company.ID
		}
	}
	return h.db.Create(&contact).Error
}

func (h *ImportHandler) insertSponsor(orgID uuid.UUID, row map[string]string) error {
	campaignIDStr := row["campaign_id"]
	if campaignIDStr == "" {
		return fmt.Errorf("campaign_id is required")
	}
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return fmt.Errorf("invalid campaign_id")
	}

	var companyID uuid.UUID
	if v := row["company_id"]; v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return fmt.Errorf("invalid company_id")
		}
		companyID = id
	} else if v := row["company_name"]; v != "" {
		var company models.Company
		if err := h.db.Where("organization_id = ? AND name = ?", orgID, v).First(&company).Error; err != nil {
			company = models.Company{OrganizationID: orgID, Name: v, Status: "active"}
			if err := h.db.Create(&company).Error; err != nil {
				return fmt.Errorf("cannot resolve company %q", v)
			}
		}
		companyID = company.ID
	} else {
		return fmt.Errorf("company_id or company_name is required")
	}

	stage := row["stage"]
	if stage == "" {
		stage = "prospect"
	}
	sponsor := models.Sponsor{
		OrganizationID: orgID,
		CampaignID:     campaignID,
		CompanyID:      companyID,
		Stage:          stage,
	}
	if v := row["tier"]; v != "" {
		sponsor.Tier = &v
	}
	if v := row["notes"]; v != "" {
		sponsor.Notes = &v
	}
	if v := row["deal_value"]; v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			sponsor.DealValue = &f
		}
	}
	if v := row["probability"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			sponsor.Probability = &n
		}
	}
	return h.db.Create(&sponsor).Error
}

func normalizeHeaders(headers []string) []string {
	out := make([]string, len(headers))
	for i, h := range headers {
		out[i] = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(h, " ", "_")))
	}
	return out
}

func mapRow(headers []string, row []string) map[string]string {
	m := make(map[string]string, len(headers))
	for i, h := range headers {
		if i < len(row) {
			m[h] = strings.TrimSpace(row[i])
		}
	}
	return m
}
