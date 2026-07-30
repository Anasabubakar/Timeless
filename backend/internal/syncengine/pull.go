package syncengine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/eventbus"
	"github.com/timeless/backend/internal/mapping"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
	"github.com/timeless/backend/internal/security"
)

// entityState fetches an internal record's current UpdatedAt (for
// conflict comparison) and applies+saves externally-sourced field values
// onto it. Deliberately bypasses the entity's own service layer (which
// would re-publish a CRUD event) — an inbound pull must never trigger
// another outbound push, or a change would ping-pong between Timeless and
// the external system forever.
type entityState struct {
	updatedAt func(ctx context.Context, db *gorm.DB, orgID, id uuid.UUID) (time.Time, bool, error)
	apply     func(ctx context.Context, db *gorm.DB, orgID, id uuid.UUID, fields map[string]interface{}) error
}

var entityStates = map[string]entityState{
	"company": {updatedAt: companyUpdatedAt, apply: applyCompany},
	"contact": {updatedAt: contactUpdatedAt, apply: applyContact},
	"sponsor": {updatedAt: sponsorUpdatedAt, apply: applySponsor},
}

func companyUpdatedAt(ctx context.Context, db *gorm.DB, orgID, id uuid.UUID) (time.Time, bool, error) {
	var c models.Company
	err := db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return time.Time{}, false, nil
	}
	return c.UpdatedAt, err == nil, err
}

func applyCompany(ctx context.Context, db *gorm.DB, orgID, id uuid.UUID, fields map[string]interface{}) error {
	var c models.Company
	if err := db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).First(&c).Error; err != nil {
		return err
	}
	mapping.ApplyToCompany(&c, fields)
	return db.WithContext(ctx).Save(&c).Error
}

func contactUpdatedAt(ctx context.Context, db *gorm.DB, orgID, id uuid.UUID) (time.Time, bool, error) {
	var c models.Contact
	err := db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return time.Time{}, false, nil
	}
	return c.UpdatedAt, err == nil, err
}

func applyContact(ctx context.Context, db *gorm.DB, orgID, id uuid.UUID, fields map[string]interface{}) error {
	var c models.Contact
	if err := db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).First(&c).Error; err != nil {
		return err
	}
	mapping.ApplyToContact(&c, fields)
	return db.WithContext(ctx).Save(&c).Error
}

func sponsorUpdatedAt(ctx context.Context, db *gorm.DB, orgID, id uuid.UUID) (time.Time, bool, error) {
	var s models.Sponsor
	err := db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return time.Time{}, false, nil
	}
	return s.UpdatedAt, err == nil, err
}

func applySponsor(ctx context.Context, db *gorm.DB, orgID, id uuid.UUID, fields map[string]interface{}) error {
	var s models.Sponsor
	if err := db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).First(&s).Error; err != nil {
		return err
	}
	mapping.ApplyToSponsor(&s, fields)
	return db.WithContext(ctx).Save(&s).Error
}

// PullService is the inbound half of the sync pipeline: given a webhook
// telling us "this external record changed," it looks up which internal
// entity it's linked to (via the SyncedEntity ledger — a record Notion
// created directly, never pushed from Timeless, has no ledger row and is
// deliberately left alone rather than guessed at), fetches the current
// remote state, and either applies it locally or — if the local side also
// changed since the last sync — flags a conflict instead of silently
// picking a winner.
type PullService struct {
	db               *gorm.DB
	cipher           *security.CredentialCipher
	fieldMappingRepo *repository.FieldMappingRepository
	integrationRepo  *repository.IntegrationRepository
	syncedRepo       *repository.SyncedEntityRepository
	historyRepo      *repository.SyncHistoryRepository
	adapters         map[string]mapping.Adapter
}

func NewPullService(
	db *gorm.DB,
	cipher *security.CredentialCipher,
	fieldMappingRepo *repository.FieldMappingRepository,
	integrationRepo *repository.IntegrationRepository,
	syncedRepo *repository.SyncedEntityRepository,
	historyRepo *repository.SyncHistoryRepository,
	adapters map[string]mapping.Adapter,
) *PullService {
	return &PullService{
		db:               db,
		cipher:           cipher,
		fieldMappingRepo: fieldMappingRepo,
		integrationRepo:  integrationRepo,
		syncedRepo:       syncedRepo,
		historyRepo:      historyRepo,
		adapters:         adapters,
	}
}

// HandleEvent is an eventbus.Handler for eventbus.NotionChanged (and, by
// the same shape, any future ExternalSystem-agnostic "changed" event) —
// evt.Data["external_system"]/["external_id"] identify the far-side
// record; evt.OrgID scopes the lookup.
func (s *PullService) HandleEvent(ctx context.Context, evt eventbus.Event) error {
	externalSystem, _ := evt.Data["external_system"].(string)
	externalID, _ := evt.Data["external_id"].(string)
	if externalSystem == "" || externalID == "" {
		return nil
	}
	orgID, err := uuid.Parse(evt.OrgID)
	if err != nil {
		return fmt.Errorf("syncengine: invalid org id %q: %w", evt.OrgID, err)
	}
	return s.PullRecord(ctx, orgID, externalSystem, externalID)
}

// PullRecord fetches externalID's current state and reconciles it against
// the internal entity it's linked to, if any.
func (s *PullService) PullRecord(ctx context.Context, orgID uuid.UUID, externalSystem, externalID string) error {
	synced, err := s.syncedRepo.FindByExternal(ctx, orgID, externalSystem, externalID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Not a record Timeless ever pushed or otherwise linked — we
		// don't know what internal entity type it should become, so we
		// leave it alone rather than guess. A user wanting Notion-first
		// creation needs a mapping-aware "adopt" flow, not implicit
		// inference from a bare webhook.
		return nil
	}
	if err != nil {
		return fmt.Errorf("syncengine: look up sync ledger: %w", err)
	}
	if synced.FieldMappingID == nil {
		return nil
	}

	fm, err := s.fieldMappingRepo.GetByID(ctx, orgID, *synced.FieldMappingID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("syncengine: load field mapping: %w", err)
	}
	if !fm.IsActive {
		return nil
	}

	integrationRec, err := s.integrationRepo.GetByID(ctx, orgID, synced.IntegrationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("syncengine: load integration: %w", err)
	}
	if integrationRec.Status != "active" {
		return nil
	}

	adapter, ok := s.adapters[externalSystem]
	if !ok {
		return nil
	}
	state, ok := entityStates[synced.EntityType]
	if !ok {
		return nil
	}

	credentials, err := s.cipher.DecryptStoredCredentials(integrationRec.Credentials)
	if err != nil {
		return fmt.Errorf("syncengine: decrypt credentials: %w", err)
	}

	remote, err := adapter.Fetch(ctx, credentials, fm.ExternalContainerID, externalID)
	if err != nil {
		s.recordFailure(ctx, synced, err)
		return fmt.Errorf("syncengine: fetch %s %s: %w", externalSystem, externalID, err)
	}

	localUpdatedAt, found, err := state.updatedAt(ctx, s.db, orgID, synced.EntityID)
	if err != nil {
		return fmt.Errorf("syncengine: load %s %s: %w", synced.EntityType, synced.EntityID, err)
	}
	if !found {
		// The internal record was deleted after the remote side changed.
		// A subsequent local delete event (if one hasn't already fired)
		// will archive the remote side via PushService — nothing to do
		// here.
		return nil
	}

	if synced.LastSyncedAt != nil && localUpdatedAt.After(*synced.LastSyncedAt) {
		// Both sides changed since the last time they agreed — resolving
		// this automatically risks silently discarding whichever side we
		// don't pick, so it goes to the conflict queue instead.
		return s.markConflict(ctx, synced, remote.LastModified)
	}

	fields, err := adapter.FromExternal(remote.RawProperties, fm)
	if err != nil {
		return fmt.Errorf("syncengine: translate %s %s from %s: %w", synced.EntityType, synced.EntityID, externalSystem, err)
	}
	if len(fields) == 0 {
		return nil
	}
	if err := state.apply(ctx, s.db, orgID, synced.EntityID, fields); err != nil {
		s.recordFailure(ctx, synced, err)
		return fmt.Errorf("syncengine: apply pulled fields to %s %s: %w", synced.EntityType, synced.EntityID, err)
	}

	now := time.Now()
	synced.SyncState = "synced"
	synced.Version++
	synced.Source = externalSystem
	synced.LastModifiedRemote = remote.LastModified
	synced.LastSyncedAt = &now
	synced.LastError = nil
	synced.ConflictState = ""
	if err := s.syncedRepo.Update(ctx, synced); err != nil {
		return fmt.Errorf("syncengine: update sync ledger: %w", err)
	}
	s.recordHistory(ctx, synced, "pulled_from_remote", nil)
	return nil
}

func (s *PullService) markConflict(ctx context.Context, synced *models.SyncedEntity, remoteModified *time.Time) error {
	synced.SyncState = "conflict"
	synced.ConflictState = "both_changed"
	synced.LastModifiedRemote = remoteModified
	if err := s.syncedRepo.Update(ctx, synced); err != nil {
		return fmt.Errorf("syncengine: mark conflict: %w", err)
	}
	s.recordHistory(ctx, synced, "conflict_detected", nil)
	return nil
}

func (s *PullService) recordFailure(ctx context.Context, synced *models.SyncedEntity, pullErr error) {
	errStr := pullErr.Error()
	synced.SyncState = "error"
	synced.LastError = &errStr
	_ = s.syncedRepo.Update(ctx, synced)
	s.recordHistory(ctx, synced, "sync_failed", pullErr)
}

func (s *PullService) recordHistory(ctx context.Context, synced *models.SyncedEntity, action string, historyErr error) {
	if synced == nil || synced.ID == uuid.Nil {
		return
	}
	h := &models.SyncHistory{
		SyncedEntityID: synced.ID,
		OrganizationID: synced.OrganizationID,
		Action:         action,
		Source:         "notion",
	}
	if historyErr != nil {
		msg := historyErr.Error()
		h.Error = &msg
	}
	_ = s.historyRepo.Record(ctx, h)
}
