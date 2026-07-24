package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

type AutomationService struct {
	repo *repository.AutomationRepository
}

func NewAutomationService(repo *repository.AutomationRepository) *AutomationService {
	return &AutomationService{repo: repo}
}

func (s *AutomationService) List(ctx context.Context, orgID uuid.UUID) ([]models.Automation, error) {
	return s.repo.List(ctx, orgID)
}

func (s *AutomationService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Automation, error) {
	return s.repo.GetByID(ctx, orgID, id)
}

func (s *AutomationService) Create(ctx context.Context, automation *models.Automation) error {
	return s.repo.Create(ctx, automation)
}

func (s *AutomationService) Update(ctx context.Context, automation *models.Automation) error {
	return s.repo.Update(ctx, automation)
}

func (s *AutomationService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return s.repo.Delete(ctx, orgID, id)
}

func (s *AutomationService) ToggleActive(ctx context.Context, orgID, id uuid.UUID, active bool) error {
	return s.repo.ToggleActive(ctx, orgID, id, active)
}
