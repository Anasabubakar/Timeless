package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

type SponsorService struct {
	repo *repository.SponsorRepository
}

func NewSponsorService(repo *repository.SponsorRepository) *SponsorService {
	return &SponsorService{repo: repo}
}

func (s *SponsorService) Create(ctx context.Context, sponsor *models.Sponsor) error {
	return s.repo.Create(ctx, sponsor)
}

func (s *SponsorService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Sponsor, error) {
	return s.repo.FindByID(ctx, orgID, id)
}

func (s *SponsorService) List(ctx context.Context, orgID uuid.UUID, limit, offset int, campaignID *uuid.UUID, stage string) ([]models.Sponsor, int64, error) {
	return s.repo.List(ctx, orgID, limit, offset, campaignID, stage)
}

func (s *SponsorService) Update(ctx context.Context, sponsor *models.Sponsor) error {
	return s.repo.Update(ctx, sponsor)
}

func (s *SponsorService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return s.repo.Delete(ctx, orgID, id)
}

func (s *SponsorService) UpdateStage(ctx context.Context, orgID, id uuid.UUID, stage string, position int) error {
	return s.repo.UpdateStage(ctx, orgID, id, stage, position)
}
