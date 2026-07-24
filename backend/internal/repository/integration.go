package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

type IntegrationRepository struct {
	db *gorm.DB
}

func NewIntegrationRepository(db *gorm.DB) *IntegrationRepository {
	return &IntegrationRepository{db: db}
}

func (r *IntegrationRepository) List(ctx context.Context, orgID uuid.UUID) ([]models.Integration, error) {
	var integrations []models.Integration
	err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("created_at DESC").Find(&integrations).Error
	return integrations, err
}

func (r *IntegrationRepository) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Integration, error) {
	var integration models.Integration
	err := r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).First(&integration).Error
	if err != nil {
		return nil, err
	}
	return &integration, nil
}

func (r *IntegrationRepository) GetByProvider(ctx context.Context, orgID uuid.UUID, provider string) (*models.Integration, error) {
	var integration models.Integration
	err := r.db.WithContext(ctx).Where("organization_id = ? AND provider = ?", orgID, provider).First(&integration).Error
	if err != nil {
		return nil, err
	}
	return &integration, nil
}

func (r *IntegrationRepository) Create(ctx context.Context, integration *models.Integration) error {
	return r.db.WithContext(ctx).Create(integration).Error
}

func (r *IntegrationRepository) Update(ctx context.Context, integration *models.Integration) error {
	return r.db.WithContext(ctx).Save(integration).Error
}

func (r *IntegrationRepository) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).Delete(&models.Integration{}).Error
}
