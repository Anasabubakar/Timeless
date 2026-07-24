package service

import (
	"github.com/google/uuid"

	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

type CommunicationService struct {
	repo *repository.CommunicationRepository
}

func NewCommunicationService(repo *repository.CommunicationRepository) *CommunicationService {
	return &CommunicationService{repo: repo}
}

func (s *CommunicationService) List(orgID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]models.Communication, int64, error) {
	return s.repo.List(orgID, filters, limit, offset)
}

func (s *CommunicationService) GetByID(orgID, id uuid.UUID) (*models.Communication, error) {
	return s.repo.GetByID(orgID, id)
}

func (s *CommunicationService) Create(comm *models.Communication) error {
	return s.repo.Create(comm)
}

func (s *CommunicationService) Update(comm *models.Communication) error {
	return s.repo.Update(comm)
}

func (s *CommunicationService) Delete(orgID, id uuid.UUID) error {
	return s.repo.Delete(orgID, id)
}

func (s *CommunicationService) GetStats(orgID uuid.UUID) (map[string]int64, error) {
	return s.repo.GetStats(orgID)
}
