package service

import (
	"context"

	"github.com/google/uuid"
)

// WebhookTrigger is a lightweight interface for triggering webhook events from anywhere in the codebase.
// Inject this into services that need to emit webhook events.
type WebhookTrigger interface {
	TriggerEvent(ctx context.Context, orgID uuid.UUID, event string, data map[string]interface{}) error
}

// Example usage in a service:
//
//	type SponsorService struct {
//	    repo    *repository.SponsorRepository
//	    webhook service.WebhookTrigger
//	}
//
//	func (s *SponsorService) UpdateStage(ctx context.Context, sponsor *models.Sponsor, newStage string) error {
//	    oldStage := sponsor.Stage
//	    sponsor.Stage = newStage
//	    if err := s.repo.Update(ctx, sponsor); err != nil {
//	        return err
//	    }
//	    // Trigger webhook event
//	    s.webhook.TriggerEvent(ctx, sponsor.OrganizationID, "sponsor.stage_changed", map[string]interface{}{
//	        "sponsor_id": sponsor.ID,
//	        "old_stage":  oldStage,
//	        "new_stage":  newStage,
//	    })
//	    return nil
//	}
