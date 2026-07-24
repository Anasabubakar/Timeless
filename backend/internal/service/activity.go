package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

type ActivityService struct {
	repo *repository.ActivityRepository
}

func NewActivityService(repo *repository.ActivityRepository) *ActivityService {
	return &ActivityService{repo: repo}
}

func (s *ActivityService) List(ctx context.Context, orgID uuid.UUID, limit, offset int, entityType, entityID, action string) ([]models.Activity, int64, error) {
	return s.repo.List(ctx, orgID, limit, offset, entityType, entityID, action)
}

func (s *ActivityService) Create(ctx context.Context, activity *models.Activity) error {
	return s.repo.Create(ctx, activity)
}

func (s *ActivityService) GetByEntityID(ctx context.Context, orgID, entityID uuid.UUID) ([]models.Activity, error) {
	return s.repo.GetByEntityID(ctx, orgID, entityID)
}
