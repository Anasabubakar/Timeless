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

// GetByExternalAccountID finds the integration a provider's webhook event
// belongs to when all we have is its own account/workspace id — there's no
// org context on an inbound webhook request otherwise.
func (r *IntegrationRepository) GetByExternalAccountID(ctx context.Context, provider, externalAccountID string) (*models.Integration, error) {
	var integration models.Integration
	err := r.db.WithContext(ctx).Where("provider = ? AND external_account_id = ?", provider, externalAccountID).First(&integration).Error
	if err != nil {
		return nil, err
	}
	return &integration, nil
}

// GetByWebhookSecret finds the integration an inbound webhook request
// belongs to by its unguessable URL token — used for providers (Zapier)
// that have no signing mechanism of their own, where the token in the URL
// path is itself the authentication.
func (r *IntegrationRepository) GetByWebhookSecret(ctx context.Context, provider, secret string) (*models.Integration, error) {
	var integration models.Integration
	err := r.db.WithContext(ctx).Where("provider = ? AND webhook_secret = ?", provider, secret).First(&integration).Error
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
