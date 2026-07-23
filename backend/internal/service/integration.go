package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/sponsoros/backend/internal/models"
	"github.com/sponsoros/backend/internal/repository"
)

type IntegrationService struct {
	repo *repository.IntegrationRepository
}

func NewIntegrationService(repo *repository.IntegrationRepository) *IntegrationService {
	return &IntegrationService{repo: repo}
}

func (s *IntegrationService) List(ctx context.Context, orgID uuid.UUID) ([]models.Integration, error) {
	return s.repo.List(ctx, orgID)
}

func (s *IntegrationService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Integration, error) {
	return s.repo.GetByID(ctx, orgID, id)
}

func (s *IntegrationService) Create(ctx context.Context, integration *models.Integration) error {
	return s.repo.Create(ctx, integration)
}

func (s *IntegrationService) Update(ctx context.Context, integration *models.Integration) error {
	return s.repo.Update(ctx, integration)
}

func (s *IntegrationService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return s.repo.Delete(ctx, orgID, id)
}
