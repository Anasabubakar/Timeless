package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/pkg/reqbind"
)

type BatchHandler struct {
	db *gorm.DB
}

func NewBatchHandler(db *gorm.DB) *BatchHandler {
	return &BatchHandler{db: db}
}

type BatchDeleteInput struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1,max=100"`
}

type BatchUpdateInput struct {
	IDs    []uuid.UUID            `json:"ids" validate:"required,min=1,max=100"`
	Fields map[string]interface{} `json:"fields" validate:"required"`
}

var allowedBatchFields = map[string]map[string]bool{
	"sponsors": {
		"stage":       true,
		"priority":    true,
		"campaign_id": true,
	},
	"contacts": {
		"status": true,
	},
	"companies": {
		"tier":     true,
		"industry": true,
	},
}

func (h *BatchHandler) BatchDelete(entity string) fiber.Handler {
	return func(c fiber.Ctx) error {
		orgID := middleware.GetOrgID(c)

		var input BatchDeleteInput
		if verr := reqbind.JSON(c, &input); verr != nil {
			return verr
		}

		result := h.db.
			Where("organization_id = ? AND id IN ?", orgID, input.IDs).
			Delete(tableModel(entity))

		if result.Error != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "batch delete failed")
		}

		return c.JSON(fiber.Map{
			"deleted": result.RowsAffected,
		})
	}
}

func (h *BatchHandler) BatchUpdate(entity string) fiber.Handler {
	return func(c fiber.Ctx) error {
		orgID := middleware.GetOrgID(c)

		var input BatchUpdateInput
		if verr := reqbind.JSON(c, &input); verr != nil {
			return verr
		}

		allowed := allowedBatchFields[entity]
		updates := make(map[string]interface{})
		for k, v := range input.Fields {
			if allowed[k] {
				updates[k] = v
			}
		}
		if len(updates) == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "no valid fields to update")
		}

		result := h.db.
			Model(tableModel(entity)).
			Where("organization_id = ? AND id IN ?", orgID, input.IDs).
			Updates(updates)

		if result.Error != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "batch update failed")
		}

		return c.JSON(fiber.Map{
			"updated": result.RowsAffected,
		})
	}
}

type tableModelType struct {
	table string
}

func (t tableModelType) TableName() string { return t.table }

func tableModel(entity string) interface{} {
	return &tableModelType{table: entity}
}
