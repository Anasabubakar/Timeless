package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/timeless/backend/internal/eventbus"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

type ContactService struct {
	repo *repository.ContactRepository
	bus  *eventbus.Bus
}

func NewContactService(repo *repository.ContactRepository) *ContactService {
	return &ContactService{repo: repo}
}

// SetBus wires event publication for this service; see CompanyService.SetBus.
func (s *ContactService) SetBus(bus *eventbus.Bus) *ContactService {
	s.bus = bus
	return s
}

func (s *ContactService) publish(ctx context.Context, eventType string, contact *models.Contact) {
	if s.bus == nil {
		return
	}
	evt := eventbus.Event{
		Type:       eventType,
		OrgID:      contact.OrganizationID.String(),
		EntityType: "contact",
		EntityID:   contact.ID.String(),
		Data:       map[string]interface{}{"first_name": contact.FirstName, "last_name": contact.LastName},
	}
	if err := s.bus.Publish(ctx, evt); err != nil {
		slog.Warn("eventbus publish failed", "event", eventType, "entity_id", contact.ID, "error", err)
	}
}

func (s *ContactService) List(ctx context.Context, orgID uuid.UUID, limit, offset int, search string) ([]models.Contact, int64, error) {
	return s.repo.List(ctx, orgID, limit, offset, search)
}

func (s *ContactService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Contact, error) {
	return s.repo.GetByID(ctx, orgID, id)
}

func (s *ContactService) Create(ctx context.Context, contact *models.Contact) error {
	if err := s.repo.Create(ctx, contact); err != nil {
		return err
	}
	s.publish(ctx, eventbus.ContactCreated, contact)
	return nil
}

func (s *ContactService) Update(ctx context.Context, contact *models.Contact) error {
	if err := s.repo.Update(ctx, contact); err != nil {
		return err
	}
	s.publish(ctx, eventbus.ContactUpdated, contact)
	return nil
}

func (s *ContactService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, orgID, id); err != nil {
		return err
	}
	s.publish(ctx, eventbus.ContactDeleted, &models.Contact{Base: models.Base{ID: id}, OrganizationID: orgID})
	return nil
}
