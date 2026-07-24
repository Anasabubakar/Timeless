package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

type OrganizationService struct {
	repo *repository.OrganizationRepository
}

func NewOrganizationService(repo *repository.OrganizationRepository) *OrganizationService {
	return &OrganizationService{repo: repo}
}

func (s *OrganizationService) GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *OrganizationService) Update(ctx context.Context, org *models.Organization) error {
	return s.repo.Update(ctx, org)
}
