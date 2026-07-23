package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sponsoros/backend/internal/models"
)

type WebhookRepository struct {
	db *gorm.DB
}

func NewWebhookRepository(db *gorm.DB) *WebhookRepository {
	return &WebhookRepository{db: db}
}

func (r *WebhookRepository) List(ctx context.Context, orgID uuid.UUID) ([]models.Webhook, error) {
	var webhooks []models.Webhook
	err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("created_at DESC").Find(&webhooks).Error
	return webhooks, err
}

func (r *WebhookRepository) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Webhook, error) {
	var webhook models.Webhook
	err := r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).First(&webhook).Error
	if err != nil {
		return nil, err
	}
	return &webhook, nil
}

func (r *WebhookRepository) GetActiveByEvent(ctx context.Context, orgID uuid.UUID, event string) ([]models.Webhook, error) {
	var webhooks []models.Webhook
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND is_active = ? AND events @> ?", orgID, true, `["`+event+`"]`).
		Find(&webhooks).Error
	return webhooks, err
}

func (r *WebhookRepository) Create(ctx context.Context, webhook *models.Webhook) error {
	return r.db.WithContext(ctx).Create(webhook).Error
}

func (r *WebhookRepository) Update(ctx context.Context, webhook *models.Webhook) error {
	return r.db.WithContext(ctx).Save(webhook).Error
}

func (r *WebhookRepository) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).Delete(&models.Webhook{}).Error
}

// Delivery log queries

func (r *WebhookRepository) ListDeliveries(ctx context.Context, orgID, webhookID uuid.UUID, limit int) ([]models.WebhookDelivery, error) {
	var deliveries []models.WebhookDelivery
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND webhook_id = ?", orgID, webhookID).
		Order("created_at DESC").
		Limit(limit).
		Find(&deliveries).Error
	return deliveries, err
}

func (r *WebhookRepository) GetDelivery(ctx context.Context, orgID, deliveryID uuid.UUID) (*models.WebhookDelivery, error) {
	var delivery models.WebhookDelivery
	err := r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, deliveryID).First(&delivery).Error
	if err != nil {
		return nil, err
	}
	return &delivery, nil
}
