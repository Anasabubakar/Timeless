package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sponsoros/backend/internal/models"
)

type AutomationRepository struct {
	db *gorm.DB
}

func NewAutomationRepository(db *gorm.DB) *AutomationRepository {
	return &AutomationRepository{db: db}
}

func (r *AutomationRepository) List(ctx context.Context, orgID uuid.UUID) ([]models.Automation, error) {
	var automations []models.Automation
	err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("created_at DESC").Find(&automations).Error
	return automations, err
}

func (r *AutomationRepository) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Automation, error) {
	var automation models.Automation
	err := r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).First(&automation).Error
	if err != nil {
		return nil, err
	}
	return &automation, nil
}

func (r *AutomationRepository) Create(ctx context.Context, automation *models.Automation) error {
	return r.db.WithContext(ctx).Create(automation).Error
}

func (r *AutomationRepository) Update(ctx context.Context, automation *models.Automation) error {
	return r.db.WithContext(ctx).Save(automation).Error
}

func (r *AutomationRepository) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).Delete(&models.Automation{}).Error
}

func (r *AutomationRepository) ToggleActive(ctx context.Context, orgID, id uuid.UUID, active bool) error {
	return r.db.WithContext(ctx).Model(&models.Automation{}).
		Where("organization_id = ? AND id = ?", orgID, id).
		Update("is_active", active).Error
}
