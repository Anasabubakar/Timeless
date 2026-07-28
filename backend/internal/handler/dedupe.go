package handler

import (
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/dedupe"
	"github.com/timeless/backend/internal/middleware"
)

// DedupeHandler exposes on-demand duplicate merging as a maintenance
// action, on top of the automatic merge that already runs after every
// sync (see worker.integrationSyncRunner).
type DedupeHandler struct {
	db *gorm.DB
}

func NewDedupeHandler(db *gorm.DB) *DedupeHandler {
	return &DedupeHandler{db: db}
}

func (h *DedupeHandler) MergeCompanies(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	summary, err := dedupe.MergeDuplicateCompanies(c.Context(), h.db, orgID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to merge duplicate companies")
	}
	return c.JSON(fiber.Map{"data": summary})
}
