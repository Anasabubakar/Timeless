package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

type SponsorRepository struct {
	db *gorm.DB
}

func NewSponsorRepository(db *gorm.DB) *SponsorRepository {
	return &SponsorRepository{db: db}
}

func (r *SponsorRepository) Create(ctx context.Context, sponsor *models.Sponsor) error {
	return r.db.WithContext(ctx).Create(sponsor).Error
}

func (r *SponsorRepository) FindByID(ctx context.Context, orgID, id uuid.UUID) (*models.Sponsor, error) {
	var sponsor models.Sponsor
	err := r.db.WithContext(ctx).
		Preload("Company").Preload("Campaign").Preload("AssignedUser").Preload("Proposals").
		First(&sponsor, "id = ? AND organization_id = ?", id, orgID).Error
	if err != nil {
		return nil, err
	}
	return &sponsor, nil
}

func (r *SponsorRepository) List(ctx context.Context, orgID uuid.UUID, limit, offset int, campaignID *uuid.UUID, stage string) ([]models.Sponsor, int64, error) {
	var sponsors []models.Sponsor
	var total int64

	q := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
	if campaignID != nil {
		q = q.Where("campaign_id = ?", *campaignID)
	}
	if stage != "" {
		q = q.Where("stage = ?", stage)
	}

	q.Model(&models.Sponsor{}).Count(&total)
	err := q.Preload("Company").Preload("AssignedUser").
		Limit(limit).Offset(offset).
		Order("position ASC, created_at DESC").
		Find(&sponsors).Error
	return sponsors, total, err
}

func (r *SponsorRepository) Update(ctx context.Context, sponsor *models.Sponsor) error {
	return r.db.WithContext(ctx).Save(sponsor).Error
}

func (r *SponsorRepository) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Delete(&models.Sponsor{}, "id = ?", id).Error
}

func (r *SponsorRepository) UpdateStage(ctx context.Context, orgID, id uuid.UUID, stage string, position int) error {
	return r.db.WithContext(ctx).
		Model(&models.Sponsor{}).
		Where("id = ? AND organization_id = ?", id, orgID).
		Updates(map[string]interface{}{
			"stage":            stage,
			"position":         position,
			"stage_entered_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *SponsorRepository) CountByStage(ctx context.Context, orgID, campaignID uuid.UUID) (map[string]int64, error) {
	type result struct {
		Stage string
		Count int64
	}
	var results []result

	err := r.db.WithContext(ctx).
		Model(&models.Sponsor{}).
		Select("stage, count(*) as count").
		Where("organization_id = ? AND campaign_id = ?", orgID, campaignID).
		Group("stage").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, r := range results {
		counts[r.Stage] = r.Count
	}
	return counts, nil
}
