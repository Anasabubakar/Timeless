package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/sponsoros/backend/internal/models"
	"github.com/sponsoros/backend/internal/repository"
)

type ContactService struct {
	repo *repository.ContactRepository
}

func NewContactService(repo *repository.ContactRepository) *ContactService {
	return &ContactService{repo: repo}
}

func (s *ContactService) List(ctx context.Context, orgID uuid.UUID, limit, offset int, search string) ([]models.Contact, int64, error) {
	return s.repo.List(ctx, orgID, limit, offset, search)
}

func (s *ContactService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Contact, error) {
	return s.repo.GetByID(ctx, orgID, id)
}

func (s *ContactService) Create(ctx context.Context, contact *models.Contact) error {
	return s.repo.Create(ctx, contact)
}

func (s *ContactService) Update(ctx context.Context, contact *models.Contact) error {
	return s.repo.Update(ctx, contact)
}

func (s *ContactService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return s.repo.Delete(ctx, orgID, id)
}
