// zapier_ingest.go turns a raw ZapierWebhookReceived event into a real
// internal record — the concrete "mapping to internal events" half of
// Zapier inbound support (the other half, PushService, already lets any
// resulting Contact flow back out to Notion automatically, since
// ContactService.Create publishes ContactCreated the same as it would for
// a contact created through the UI).
package syncengine

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/eventbus"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/normalize"
	"github.com/timeless/backend/internal/repository"
	"github.com/timeless/backend/internal/service"
)

// zapierContactEventTypes are the evt.Data["event_type"] values a user's
// Zap can send that this processor understands. Anything else is left
// alone — evt.Data is still available in SyncHistory/logs for a future
// event type to pick up, this just doesn't guess at unknown shapes.
var zapierContactEventTypes = map[string]bool{
	"contact": true,
	"lead":    true,
}

// ZapierIngestService is the default inbound processor for Zapier
// webhooks: "new lead/contact" is by far the most common Zap trigger
// pattern (form submission, chat capture, another CRM's webhook), so this
// is the one payload shape handled out of the box. It expects the user's
// Zap to send JSON shaped like:
//
//	{"event_type": "contact", "email": "...", "first_name": "...", "last_name": "...", "company_name": "..."}
//
// documented in the integrations dashboard alongside the generated
// webhook URL.
type ZapierIngestService struct {
	contactRepo *repository.ContactRepository
	companyRepo *repository.CompanyRepository
	contactSvc  *service.ContactService
}

func NewZapierIngestService(contactRepo *repository.ContactRepository, companyRepo *repository.CompanyRepository, contactSvc *service.ContactService) *ZapierIngestService {
	return &ZapierIngestService{contactRepo: contactRepo, companyRepo: companyRepo, contactSvc: contactSvc}
}

// HandleEvent is an eventbus.Handler for eventbus.ZapierWebhookReceived.
func (s *ZapierIngestService) HandleEvent(ctx context.Context, evt eventbus.Event) error {
	eventType, _ := evt.Data["event_type"].(string)
	if !zapierContactEventTypes[eventType] {
		return nil
	}
	orgID, err := uuid.Parse(evt.OrgID)
	if err != nil {
		return fmt.Errorf("syncengine: invalid org id %q: %w", evt.OrgID, err)
	}

	email := normalize.Email(stringField(evt.Data, "email"))
	firstName := stringField(evt.Data, "first_name")
	lastName := stringField(evt.Data, "last_name")
	if email == "" && firstName == "" && lastName == "" {
		return nil // not enough to act on — nothing worth creating
	}

	if email != "" {
		if existing, err := s.contactRepo.FindByEmail(ctx, orgID, email); err == nil {
			return s.updateExisting(ctx, existing, evt.Data)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("syncengine: look up contact by email: %w", err)
		}
	}

	contact := &models.Contact{
		OrganizationID: orgID,
		FirstName:      firstName,
		LastName:       lastName,
		Status:         "active",
	}
	if email != "" {
		contact.Email = &email
	}
	if phone := stringField(evt.Data, "phone"); phone != "" {
		contact.Phone = &phone
	}
	if title := stringField(evt.Data, "title"); title != "" {
		contact.Title = &title
	}
	if companyName := stringField(evt.Data, "company_name"); companyName != "" {
		company, err := s.findOrCreateCompany(ctx, orgID, companyName)
		if err != nil {
			return fmt.Errorf("syncengine: find or create company %q: %w", companyName, err)
		}
		contact.CompanyID = &company.ID
	}

	// Through the service (not the repo) so this publishes ContactCreated
	// like any other creation path — that's what lets PushService pick it
	// up and sync it to Notion without this package needing to know
	// anything about Notion.
	return s.contactSvc.Create(ctx, contact)
}

func (s *ZapierIngestService) updateExisting(ctx context.Context, existing *models.Contact, data map[string]interface{}) error {
	changed := false
	if phone := stringField(data, "phone"); phone != "" && (existing.Phone == nil || *existing.Phone != phone) {
		existing.Phone = &phone
		changed = true
	}
	if title := stringField(data, "title"); title != "" && (existing.Title == nil || *existing.Title != title) {
		existing.Title = &title
		changed = true
	}
	if !changed {
		return nil
	}
	return s.contactSvc.Update(ctx, existing)
}

func (s *ZapierIngestService) findOrCreateCompany(ctx context.Context, orgID uuid.UUID, name string) (*models.Company, error) {
	normName := normalize.CompanyName(name)
	existing, err := s.companyRepo.FindByName(ctx, orgID, normName)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	company := &models.Company{OrganizationID: orgID, Name: name, Status: "active"}
	if err := s.companyRepo.Create(ctx, company); err != nil {
		return nil, err
	}
	return company, nil
}

func stringField(data map[string]interface{}, key string) string {
	v, _ := data[key].(string)
	return v
}
