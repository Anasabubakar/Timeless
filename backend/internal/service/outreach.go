package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/sponsoros/backend/internal/models"
	"github.com/sponsoros/backend/internal/repository"
)

type OutreachService struct {
	repo *repository.OutreachRepository
}

func NewOutreachService(repo *repository.OutreachRepository) *OutreachService {
	return &OutreachService{repo: repo}
}

func (s *OutreachService) ListSequences(ctx context.Context, orgID uuid.UUID, status string) ([]models.OutreachSequence, error) {
	return s.repo.ListSequences(ctx, orgID, status)
}

func (s *OutreachService) GetSequence(ctx context.Context, orgID, id uuid.UUID) (*models.OutreachSequence, error) {
	return s.repo.GetSequence(ctx, orgID, id)
}

func (s *OutreachService) CreateSequence(ctx context.Context, seq *models.OutreachSequence) error {
	return s.repo.CreateSequence(ctx, seq)
}

func (s *OutreachService) UpdateSequence(ctx context.Context, seq *models.OutreachSequence) error {
	return s.repo.UpdateSequence(ctx, seq)
}

func (s *OutreachService) DeleteSequence(ctx context.Context, orgID, id uuid.UUID) error {
	return s.repo.DeleteSequence(ctx, orgID, id)
}

func (s *OutreachService) ListEnrollments(ctx context.Context, orgID uuid.UUID, sequenceID *uuid.UUID, status string) ([]models.Enrollment, int64, error) {
	return s.repo.ListEnrollments(ctx, orgID, sequenceID, status)
}

func (s *OutreachService) CreateEnrollment(ctx context.Context, enrollment *models.Enrollment) error {
	return s.repo.CreateEnrollment(ctx, enrollment)
}
