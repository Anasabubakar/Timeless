package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/timeless/backend/internal/eventbus"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

type SponsorService struct {
	repo *repository.SponsorRepository
	bus  *eventbus.Bus
}

func NewSponsorService(repo *repository.SponsorRepository) *SponsorService {
	return &SponsorService{repo: repo}
}

// SetBus wires event publication for this service; see CompanyService.SetBus.
func (s *SponsorService) SetBus(bus *eventbus.Bus) *SponsorService {
	s.bus = bus
	return s
}

func (s *SponsorService) publish(ctx context.Context, eventType string, sponsor *models.Sponsor) {
	if s.bus == nil {
		return
	}
	evt := eventbus.Event{
		Type:       eventType,
		OrgID:      sponsor.OrganizationID.String(),
		EntityType: "sponsor",
		EntityID:   sponsor.ID.String(),
		Data:       map[string]interface{}{"stage": sponsor.Stage, "campaign_id": sponsor.CampaignID.String(), "company_id": sponsor.CompanyID.String()},
	}
	if err := s.bus.Publish(ctx, evt); err != nil {
		slog.Warn("eventbus publish failed", "event", eventType, "entity_id", sponsor.ID, "error", err)
	}
}

func (s *SponsorService) Create(ctx context.Context, sponsor *models.Sponsor) error {
	if err := s.repo.Create(ctx, sponsor); err != nil {
		return err
	}
	s.publish(ctx, eventbus.SponsorCreated, sponsor)
	return nil
}

func (s *SponsorService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Sponsor, error) {
	return s.repo.FindByID(ctx, orgID, id)
}

func (s *SponsorService) List(ctx context.Context, orgID uuid.UUID, limit, offset int, campaignID *uuid.UUID, stage string) ([]models.Sponsor, int64, error) {
	return s.repo.List(ctx, orgID, limit, offset, campaignID, stage)
}

func (s *SponsorService) Update(ctx context.Context, sponsor *models.Sponsor) error {
	if err := s.repo.Update(ctx, sponsor); err != nil {
		return err
	}
	s.publish(ctx, eventbus.SponsorUpdated, sponsor)
	return nil
}

func (s *SponsorService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, orgID, id); err != nil {
		return err
	}
	s.publish(ctx, eventbus.SponsorDeleted, &models.Sponsor{Base: models.Base{ID: id}, OrganizationID: orgID})
	return nil
}

func (s *SponsorService) UpdateStage(ctx context.Context, orgID, id uuid.UUID, stage string, position int) error {
	if err := s.repo.UpdateStage(ctx, orgID, id, stage, position); err != nil {
		return err
	}
	sponsor, err := s.repo.FindByID(ctx, orgID, id)
	if err == nil {
		s.publish(ctx, eventbus.SponsorUpdated, sponsor)
	}
	return nil
}
