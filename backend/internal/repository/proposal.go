package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sponsoros/backend/internal/models"
)

type ProposalRepository struct {
	db *gorm.DB
}

func NewProposalRepository(db *gorm.DB) *ProposalRepository {
	return &ProposalRepository{db: db}
}

func (r *ProposalRepository) List(ctx context.Context, orgID uuid.UUID, sponsorID *uuid.UUID, status string) ([]models.Proposal, int64, error) {
	var proposals []models.Proposal
	var total int64

	q := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
	if sponsorID != nil {
		q = q.Where("sponsor_id = ?", *sponsorID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}

	q.Model(&models.Proposal{}).Count(&total)
	err := q.Order("created_at DESC").Find(&proposals).Error
	return proposals, total, err
}

func (r *ProposalRepository) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Proposal, error) {
	var proposal models.Proposal
	err := r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).First(&proposal).Error
	if err != nil {
		return nil, err
	}
	return &proposal, nil
}

func (r *ProposalRepository) Create(ctx context.Context, proposal *models.Proposal) error {
	return r.db.WithContext(ctx).Create(proposal).Error
}

func (r *ProposalRepository) Update(ctx context.Context, proposal *models.Proposal) error {
	return r.db.WithContext(ctx).Save(proposal).Error
}

func (r *ProposalRepository) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).Delete(&models.Proposal{}).Error
}
