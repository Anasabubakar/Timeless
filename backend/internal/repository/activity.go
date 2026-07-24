package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

type ActivityRepository struct {
	db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

func (r *ActivityRepository) List(ctx context.Context, orgID uuid.UUID, limit, offset int, entityType, entityID, action string) ([]models.Activity, int64, error) {
	var activities []models.Activity
	var total int64

	q := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
	if entityType != "" {
		q = q.Where("entity_type = ?", entityType)
	}
	if entityID != "" {
		q = q.Where("entity_id = ?", entityID)
	}
	if action != "" {
		q = q.Where("action = ?", action)
	}

	q.Model(&models.Activity{}).Count(&total)
	err := q.Preload("User").Order("created_at DESC").Limit(limit).Offset(offset).Find(&activities).Error
	return activities, total, err
}

func (r *ActivityRepository) Create(ctx context.Context, activity *models.Activity) error {
	return r.db.WithContext(ctx).Create(activity).Error
}

func (r *ActivityRepository) GetByEntityID(ctx context.Context, orgID, entityID uuid.UUID) ([]models.Activity, error) {
	var activities []models.Activity
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND entity_id = ?", orgID, entityID).
		Preload("User").
		Order("created_at DESC").
		Find(&activities).Error
	return activities, err
}
