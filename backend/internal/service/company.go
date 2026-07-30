package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/timeless/backend/internal/eventbus"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

type CompanyService struct {
	repo *repository.CompanyRepository
	bus  *eventbus.Bus
}

func NewCompanyService(repo *repository.CompanyRepository) *CompanyService {
	return &CompanyService{repo: repo}
}

// SetBus wires event publication for this service. Optional — an unwired
// service behaves exactly as before (no events, no sync). Calling this is
// how router.Setup opts a service into the sync pipeline without every
// existing caller/test needing to thread a bus through the constructor.
func (s *CompanyService) SetBus(bus *eventbus.Bus) *CompanyService {
	s.bus = bus
	return s
}

// publish is best-effort: a queueing failure (e.g. Redis blip) must not
// fail an otherwise-successful write. The periodic resync worker is the
// fallback path for anything missed this way.
func (s *CompanyService) publish(ctx context.Context, eventType string, company *models.Company) {
	if s.bus == nil {
		return
	}
	evt := eventbus.Event{
		Type:       eventType,
		OrgID:      company.OrganizationID.String(),
		EntityType: "company",
		EntityID:   company.ID.String(),
		Data:       map[string]interface{}{"name": company.Name},
	}
	if err := s.bus.Publish(ctx, evt); err != nil {
		slog.Warn("eventbus publish failed", "event", eventType, "entity_id", company.ID, "error", err)
	}
}

func (s *CompanyService) Create(ctx context.Context, company *models.Company) error {
	if err := s.repo.Create(ctx, company); err != nil {
		return err
	}
	s.publish(ctx, eventbus.CompanyCreated, company)
	return nil
}

func (s *CompanyService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Company, error) {
	return s.repo.FindByID(ctx, orgID, id)
}

func (s *CompanyService) List(ctx context.Context, orgID uuid.UUID, limit, offset int, search string) ([]models.Company, int64, error) {
	return s.repo.List(ctx, orgID, limit, offset, search)
}

func (s *CompanyService) Update(ctx context.Context, company *models.Company) error {
	if err := s.repo.Update(ctx, company); err != nil {
		return err
	}
	s.publish(ctx, eventbus.CompanyUpdated, company)
	return nil
}

func (s *CompanyService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, orgID, id); err != nil {
		return err
	}
	s.publish(ctx, eventbus.CompanyDeleted, &models.Company{Base: models.Base{ID: id}, OrganizationID: orgID})
	return nil
}
