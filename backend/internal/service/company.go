package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

type CompanyService struct {
	repo *repository.CompanyRepository
}

func NewCompanyService(repo *repository.CompanyRepository) *CompanyService {
	return &CompanyService{repo: repo}
}

func (s *CompanyService) Create(ctx context.Context, company *models.Company) error {
	return s.repo.Create(ctx, company)
}

func (s *CompanyService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Company, error) {
	return s.repo.FindByID(ctx, orgID, id)
}

func (s *CompanyService) List(ctx context.Context, orgID uuid.UUID, limit, offset int, search string) ([]models.Company, int64, error) {
	return s.repo.List(ctx, orgID, limit, offset, search)
}

func (s *CompanyService) Update(ctx context.Context, company *models.Company) error {
	return s.repo.Update(ctx, company)
}

func (s *CompanyService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return s.repo.Delete(ctx, orgID, id)
}
