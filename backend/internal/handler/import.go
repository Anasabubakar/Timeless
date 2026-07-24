package handler

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
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

func (h *ImportHandler) Import(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	entity := c.Query("entity", "sponsors")
	if e := c.Locals("entity"); e != nil {
		entity = e.(string)
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "file upload required: "+err.Error())
	}

	file, err := fileHeader.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to open file")
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil && err != io.EOF {
		return fiber.NewError(fiber.StatusBadRequest, "failed to parse csv: "+err.Error())
	}
	if len(records) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "csv file is empty")
	}

	headers := records[0]
	dataRows := records[1:]

	var imported int
	var errors []string

	switch entity {
	case "sponsors":
		imported, errors = importSponsors(c.Context(), h.db, orgID, headers, dataRows)
	case "contacts":
		imported, errors = importContacts(c.Context(), h.db, orgID, headers, dataRows)
	case "companies":
		imported, errors = importCompanies(c.Context(), h.db, orgID, headers, dataRows)
	default:
		return fiber.NewError(fiber.StatusBadRequest, "invalid entity: "+entity)
	}

	return c.JSON(fiber.Map{
		"imported": imported,
		"errors":   errors,
	})
}

func importSponsors(ctx context.Context, db *gorm.DB, orgID uuid.UUID, headers []string, rows [][]string) (int, []string) {
	imported := 0
	var errors []string

	hMap := make(map[string]int)
	for i, h := range headers {
		hMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	batchSize := 50
	batch := make([]models.Sponsor, 0, batchSize)

	flushBatch := func() {
		if len(batch) == 0 {
			return
		}
		res := db.WithContext(ctx).CreateInBatches(batch, batchSize)
		if res.Error != nil {
			errors = append(errors, fmt.Sprintf("batch create error: %v", res.Error))
		} else {
			imported += len(batch)
		}
		batch = batch[:0]
	}

	for rowIdx, row := range rows {
		line := rowIdx + 2
		s := models.Sponsor{
			OrganizationID: orgID,
			Stage:          "prospect",
			Position:       0,
		}

		if v, ok := hMap["company_id"]; ok && v < len(row) {
			val := strings.TrimSpace(row[v])
			if val != "" {
				if id, err := uuid.Parse(val); err == nil {
					s.CompanyID = id
				} else {
					errors = append(errors, fmt.Sprintf("row %d: invalid company_id %q", line, val))
				}
			}
		}

		if v, ok := hMap["campaign_id"]; ok && v < len(row) {
			val := strings.TrimSpace(row[v])
			if val != "" {
				if id, err := uuid.Parse(val); err == nil {
					s.CampaignID = id
				} else {
					errors = append(errors, fmt.Sprintf("row %d: invalid campaign_id %q", line, val))
				}
			} else {
				s.CampaignID = uuid.UUID{}
			}
		}

		if v, ok := hMap["stage"]; ok && v < len(row) {
			val := strings.TrimSpace(row[v])
			if val != "" {
				s.Stage = val
			}
		}

		if v, ok := hMap["tier"]; ok && v < len(row) {
			val := strings.TrimSpace(row[v])
			if val != "" {
				s.Tier = &val
			}
		}

		if v, ok := hMap["deal_value"]; ok && v < len(row) {
			val := strings.TrimSpace(row[v])
			if val != "" {
				var f float64
				_, parseErr := fmt.Sscanf(val, "%f", &f)
				if parseErr == nil {
					s.DealValue = &f
				} else {
					errors = append(errors, fmt.Sprintf("row %d: invalid deal_value %q", line, val))
				}
			}
		}

		if s.CompanyID == uuid.Nil {
			errors = append(errors, fmt.Sprintf("row %d: company_id is required", line))
			continue
		}

		batch = append(batch, s)
		if len(batch) >= batchSize {
			flushBatch()
		}
	}

	flushBatch()
	return imported, errors
}

func importContacts(ctx context.Context, db *gorm.DB, orgID uuid.UUID, headers []string, rows [][]string) (int, []string) {
	imported := 0
	var errors []string

	hMap := make(map[string]int)
	for i, h := range headers {
		hMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	batch := make([]models.Contact, 0, 50)

	flushBatch := func() {
		if len(batch) == 0 {
			return
		}
		res := db.WithContext(ctx).CreateInBatches(batch, 50)
		if res.Error != nil {
			errors = append(errors, fmt.Sprintf("batch create error: %v", res.Error))
		} else {
			imported += len(batch)
		}
		batch = batch[:0]
	}

	for rowIdx, row := range rows {
		line := rowIdx + 2
		c := models.Contact{
			OrganizationID: orgID,
			Status:         "active",
		}

		if v, ok := hMap["first_name"]; ok && v < len(row) {
			c.FirstName = strings.TrimSpace(row[v])
		}
		if c.FirstName == "" {
			errors = append(errors, fmt.Sprintf("row %d: first_name is required", line))
			continue
		}

		if v, ok := hMap["last_name"]; ok && v < len(row) {
			c.LastName = strings.TrimSpace(row[v])
		}
		if c.LastName == "" {
			errors = append(errors, fmt.Sprintf("row %d: last_name is required", line))
			continue
		}

		if v, ok := hMap["email"]; ok && v < len(row) {
			val := strings.TrimSpace(row[v])
			if val != "" {
				c.Email = &val
			}
		}

		if v, ok := hMap["phone"]; ok && v < len(row) {
			val := strings.TrimSpace(row[v])
			if val != "" {
				c.Phone = &val
			}
		}

		if v, ok := hMap["title"]; ok && v < len(row) {
			val := strings.TrimSpace(row[v])
			if val != "" {
				c.Title = &val
			}
		}

		if v, ok := hMap["company_id"]; ok && v < len(row) {
			val := strings.TrimSpace(row[v])
			if val != "" {
				if id, err := uuid.Parse(val); err == nil {
					c.CompanyID = &id
				} else {
					errors = append(errors, fmt.Sprintf("row %d: invalid company_id %q", line, val))
				}
			}
		}

		batch = append(batch, c)
		if len(batch) >= 50 {
			flushBatch()
		}
	}

	flushBatch()
	return imported, errors
}

func importCompanies(ctx context.Context, db *gorm.DB, orgID uuid.UUID, headers []string, rows [][]string) (int, []string) {
	imported := 0
	var errors []string

	hMap := make(map[string]int)
	for i, h := range headers {
		hMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	batch := make([]models.Company, 0, 50)

	flushBatch := func() {
		if len(batch) == 0 {
			return
		}
		res := db.WithContext(ctx).CreateInBatches(batch, 50)
		if res.Error != nil {
			errors = append(errors, fmt.Sprintf("batch create error: %v", res.Error))
		} else {
			imported += len(batch)
		}
		batch = batch[:0]
	}

	for rowIdx, row := range rows {
		line := rowIdx + 2
		company := models.Company{
			OrganizationID: orgID,
			Status:         "active",
		}

		if v, ok := hMap["name"]; ok && v < len(row) {
			company.Name = strings.TrimSpace(row[v])
		}
		if company.Name == "" {
			errors = append(errors, fmt.Sprintf("row %d: name is required", line))
			continue
		}

		if v, ok := hMap["domain"]; ok && v < len(row) {
			val := strings.TrimSpace(row[v])
			if val != "" {
				company.Domain = &val
			}
		}

		if v, ok := hMap["website"]; ok && v < len(row) {
			val := strings.TrimSpace(row[v])
			if val != "" {
				company.Website = &val
			}
		}

		if v, ok := hMap["industry_id"]; ok && v < len(row) {
			val := strings.TrimSpace(row[v])
			if val != "" {
				if id, err := uuid.Parse(val); err == nil {
					company.IndustryID = &id
				} else {
					errors = append(errors, fmt.Sprintf("row %d: invalid industry_id %q", line, val))
				}
			}
		}

		batch = append(batch, company)
		if len(batch) >= 50 {
			flushBatch()
		}
	}

	flushBatch()
	return imported, errors
}
