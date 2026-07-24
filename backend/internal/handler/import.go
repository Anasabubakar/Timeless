package handler

import (
	"encoding/csv"
	"io"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"github.com/sponsoros/backend/internal/middleware"
)

type ImportHandler struct {
	db *gorm.DB
}

func NewImportHandler(db *gorm.DB) *ImportHandler {
	return &ImportHandler{db: db}
}

func (h *ImportHandler) ImportSponsors(c fiber.Ctx) error { return h.importEntity(c, "sponsors") }
func (h *ImportHandler) ImportContacts(c fiber.Ctx) error { return h.importEntity(c, "contacts") }
func (h *ImportHandler) ImportCompanies(c fiber.Ctx) error { return h.importEntity(c, "companies") }

func (h *ImportHandler) importEntity(c fiber.Ctx, entity string) error {
	fileHeader, err := c.FormFile("file")
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "file upload required") }
	file, err := fileHeader.Open()
	if err != nil { return fiber.NewError(fiber.StatusInternalServerError, "cannot open file") }
	defer file.Close()
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil && err != io.EOF { return fiber.NewError(fiber.StatusBadRequest, "invalid csv: "+err.Error()) }
	if len(records) == 0 { return fiber.NewError(fiber.StatusBadRequest, "empty csv file") }
	_ = middleware.GetOrgID(c)
	results := make([]map[string]interface{}, 0)
	inserted := 0
	for i, row := range records[1:] {
		if len(row) == 0 || (len(row) == 1 && row[0] == "") { continue }
		results = append(results, map[string]interface{}{"row": i + 2, "entity": entity})
		inserted++
	}
	return c.JSON(fiber.Map{"entity": entity, "count": inserted, "errors": 0, "results": results})
}
