package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/sponsoros/backend/internal/models"
	"github.com/sponsoros/backend/internal/repository"
)

type CampaignService struct {
	repo *repository.CampaignRepository
}

func NewCampaignService(repo *repository.CampaignRepository) *CampaignService {
	return &CampaignService{repo: repo}
}

func (s *CampaignService) Create(ctx context.Context, campaign *models.Campaign) error {
	return s.repo.Create(ctx, campaign)
}

func (s *CampaignService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Campaign, error) {
	return s.repo.FindByID(ctx, orgID, id)
}

func (s *CampaignService) List(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]models.Campaign, int64, error) {
	return s.repo.List(ctx, orgID, limit, offset)
}

func (s *CampaignService) Update(ctx context.Context, campaign *models.Campaign) error {
	return s.repo.Update(ctx, campaign)
}

func (s *CampaignService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return s.repo.Delete(ctx, orgID, id)
}
