package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

type FieldMappingRepository struct {
	db *gorm.DB
}

func NewFieldMappingRepository(db *gorm.DB) *FieldMappingRepository {
	return &FieldMappingRepository{db: db}
}

func (r *FieldMappingRepository) Create(ctx context.Context, fm *models.FieldMapping) error {
	return r.db.WithContext(ctx).Create(fm).Error
}

func (r *FieldMappingRepository) Update(ctx context.Context, fm *models.FieldMapping) error {
	return r.db.WithContext(ctx).Save(fm).Error
}

func (r *FieldMappingRepository) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).Delete(&models.FieldMapping{}).Error
}

func (r *FieldMappingRepository) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.FieldMapping, error) {
	var fm models.FieldMapping
	if err := r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).First(&fm).Error; err != nil {
		return nil, err
	}
	return &fm, nil
}

func (r *FieldMappingRepository) ListByIntegration(ctx context.Context, orgID, integrationID uuid.UUID) ([]models.FieldMapping, error) {
	var mappings []models.FieldMapping
	err := r.db.WithContext(ctx).Where("organization_id = ? AND integration_id = ?", orgID, integrationID).Find(&mappings).Error
	return mappings, err
}

// ListActiveByEntityType is the write-back path's lookup: "which active
// mappings (across every connected integration) exist for this internal
// entity type in this org?" — a company might sync to Notion and, later,
// HubSpot simultaneously, so this deliberately returns every match rather
// than assuming one mapping per entity type.
func (r *FieldMappingRepository) ListActiveByEntityType(ctx context.Context, orgID uuid.UUID, entityType string) ([]models.FieldMapping, error) {
	var mappings []models.FieldMapping
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND entity_type = ? AND is_active = ?", orgID, entityType, true).
		Find(&mappings).Error
	return mappings, err
}
