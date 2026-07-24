package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

type CampaignRepository struct {
	db *gorm.DB
}

func NewCampaignRepository(db *gorm.DB) *CampaignRepository {
	return &CampaignRepository{db: db}
}

func (r *CampaignRepository) Create(ctx context.Context, campaign *models.Campaign) error {
	return r.db.WithContext(ctx).Create(campaign).Error
}

func (r *CampaignRepository) FindByID(ctx context.Context, orgID, id uuid.UUID) (*models.Campaign, error) {
	var campaign models.Campaign
	err := r.db.WithContext(ctx).
		Preload("Sponsors").Preload("Project").
		First(&campaign, "id = ? AND organization_id = ?", id, orgID).Error
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

func (r *CampaignRepository) List(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]models.Campaign, int64, error) {
	var campaigns []models.Campaign
	var total int64

	q := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
	q.Model(&models.Campaign{}).Count(&total)
	err := q.Preload("Project").
		Limit(limit).Offset(offset).
		Order("created_at DESC").
		Find(&campaigns).Error
	return campaigns, total, err
}

func (r *CampaignRepository) Update(ctx context.Context, campaign *models.Campaign) error {
	return r.db.WithContext(ctx).Save(campaign).Error
}

func (r *CampaignRepository) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Delete(&models.Campaign{}, "id = ?", id).Error
}
