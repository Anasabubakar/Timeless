// Package syncengine wires the mapping engine (internal/mapping) and the
// event bus (internal/eventbus) together: when an entity service publishes
// a CRUD event, PushService looks up every active FieldMapping for that
// org+entity type, resolves the right Adapter for each mapping's
// integration, and pushes the current state of the record — creating a
// SyncedEntity ledger row on first sync, updating it on every one after.
package syncengine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/eventbus"
	"github.com/timeless/backend/internal/integration"
	"github.com/timeless/backend/internal/mapping"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
	"github.com/timeless/backend/internal/security"
)

// entityLoader fetches the current state of one internal record and
// converts it to a SyncableRecord. found=false means the record no longer
// exists (a delete event, or a race where it was deleted after the event
// was enqueued) — either way, the caller should archive the external
// side rather than push stale/empty fields.
type entityLoader func(ctx context.Context, db *gorm.DB, orgID, id uuid.UUID) (mapping.SyncableRecord, bool, error)

var entityLoaders = map[string]entityLoader{
	"company": loadCompany,
	"contact": loadContact,
	"sponsor": loadSponsor,
}

func loadCompany(ctx context.Context, db *gorm.DB, orgID, id uuid.UUID) (mapping.SyncableRecord, bool, error) {
	var c models.Company
	err := db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return mapping.SyncableRecord{}, false, nil
	}
	if err != nil {
		return mapping.SyncableRecord{}, false, err
	}
	return mapping.CompanyToRecord(&c), true, nil
}

func loadContact(ctx context.Context, db *gorm.DB, orgID, id uuid.UUID) (mapping.SyncableRecord, bool, error) {
	var c models.Contact
	err := db.WithContext(ctx).Where("organization_id = ? AND id = ?", orgID, id).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return mapping.SyncableRecord{}, false, nil
	}
	if err != nil {
		return mapping.SyncableRecord{}, false, err
	}
	return mapping.ContactToRecord(&c), true, nil
}

func loadSponsor(ctx context.Context, db *gorm.DB, orgID, id uuid.UUID) (mapping.SyncableRecord, bool, error) {
	var s models.Sponsor
	err := db.WithContext(ctx).Preload("Company").Where("organization_id = ? AND id = ?", orgID, id).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return mapping.SyncableRecord{}, false, nil
	}
	if err != nil {
		return mapping.SyncableRecord{}, false, err
	}
	return mapping.SponsorToRecord(&s), true, nil
}

// PushService is the outbound half of the sync pipeline — see package doc.
type PushService struct {
	db               *gorm.DB
	cipher           *security.CredentialCipher
	fieldMappingRepo *repository.FieldMappingRepository
	integrationRepo  *repository.IntegrationRepository
	syncedRepo       *repository.SyncedEntityRepository
	historyRepo      *repository.SyncHistoryRepository
	adapters         map[string]mapping.Adapter
}

func NewPushService(
	db *gorm.DB,
	cipher *security.CredentialCipher,
	fieldMappingRepo *repository.FieldMappingRepository,
	integrationRepo *repository.IntegrationRepository,
	syncedRepo *repository.SyncedEntityRepository,
	historyRepo *repository.SyncHistoryRepository,
	adapters map[string]mapping.Adapter,
) *PushService {
	return &PushService{
		db:               db,
		cipher:           cipher,
		fieldMappingRepo: fieldMappingRepo,
		integrationRepo:  integrationRepo,
		syncedRepo:       syncedRepo,
		historyRepo:      historyRepo,
		adapters:         adapters,
	}
}

// HandleEvent is an eventbus.Handler — wire it with bus.Subscribe for
// every CRUD event type of every syncable entity. Errors returned here
// cause the asynq task carrying this dispatch to retry (see
// worker.HandleEventDispatch), so a transient provider failure is
// automatically retried rather than silently dropped.
func (s *PushService) HandleEvent(ctx context.Context, evt eventbus.Event) error {
	if _, ok := entityLoaders[evt.EntityType]; !ok {
		return nil // not a syncable entity type — nothing to do
	}
	orgID, err := uuid.Parse(evt.OrgID)
	if err != nil {
		return fmt.Errorf("syncengine: invalid org id %q: %w", evt.OrgID, err)
	}
	entityID, err := uuid.Parse(evt.EntityID)
	if err != nil {
		return fmt.Errorf("syncengine: invalid entity id %q: %w", evt.EntityID, err)
	}

	mappings, err := s.fieldMappingRepo.ListActiveByEntityType(ctx, orgID, evt.EntityType)
	if err != nil {
		return fmt.Errorf("syncengine: list field mappings: %w", err)
	}
	if len(mappings) == 0 {
		return nil
	}

	isDelete := strings.HasSuffix(evt.Type, "Deleted")

	var errs []error
	for i := range mappings {
		if err := s.pushOne(ctx, orgID, evt.EntityType, entityID, &mappings[i], isDelete); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("syncengine: %d of %d mapping(s) failed: %w", len(errs), len(mappings), errs[0])
	}
	return nil
}

func (s *PushService) pushOne(ctx context.Context, orgID uuid.UUID, entityType string, entityID uuid.UUID, fm *models.FieldMapping, isDelete bool) error {
	integrationRec, err := s.integrationRepo.GetByID(ctx, orgID, fm.IntegrationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // mapping points at an integration that no longer exists
		}
		return fmt.Errorf("load integration %s: %w", fm.IntegrationID, err)
	}
	if integrationRec.Status != "active" {
		return nil // don't push against a disconnected/expired integration
	}
	adapter, ok := s.adapters[integrationRec.Provider]
	if !ok {
		return nil // no adapter registered for this provider yet
	}

	credentials, err := s.cipher.DecryptStoredCredentials(integrationRec.Credentials)
	if err != nil {
		return fmt.Errorf("decrypt credentials for integration %s: %w", integrationRec.ID, err)
	}

	synced, err := s.syncedRepo.FindByInternal(ctx, orgID, entityType, entityID, adapter.System())
	notFoundLocally := errors.Is(err, gorm.ErrRecordNotFound)
	if err != nil && !notFoundLocally {
		return fmt.Errorf("look up sync ledger: %w", err)
	}

	if isDelete {
		if notFoundLocally || synced.ExternalID == "" {
			return nil // never synced — nothing external to remove
		}
		if err := adapter.Archive(ctx, credentials, synced.ExternalID); err != nil {
			s.recordFailure(ctx, synced, err)
			return fmt.Errorf("archive %s %s: %w", adapter.System(), synced.ExternalID, err)
		}
		now := time.Now()
		synced.SyncState = "synced"
		synced.Version++
		synced.Source = "timeless"
		synced.LastModifiedLocal = &now
		synced.LastSyncedAt = &now
		synced.LastError = nil
		_ = s.syncedRepo.Update(ctx, synced)
		s.recordHistory(ctx, synced, "pushed_to_remote", nil)
		return nil
	}

	loader := entityLoaders[entityType]
	record, found, err := loader(ctx, s.db, orgID, entityID)
	if err != nil {
		return fmt.Errorf("load %s %s: %w", entityType, entityID, err)
	}
	if !found {
		// The record was deleted after this event was enqueued (or a
		// non-delete event raced a delete). Treat it the same as an
		// explicit delete rather than pushing empty fields.
		return s.pushOne(ctx, orgID, entityType, entityID, fm, true)
	}

	properties, err := adapter.ToExternal(record, fm)
	if err != nil {
		return fmt.Errorf("translate %s %s to %s: %w", entityType, entityID, adapter.System(), err)
	}

	existingExternalID := ""
	expectedRemoteVersion := ""
	if !notFoundLocally {
		existingExternalID = synced.ExternalID
		// Only set once we've actually observed the remote side (via an
		// inbound pull) — until then there's nothing to conflict-check
		// against, and the adapter treats an empty expected version as
		// "skip the check," which is correct for a first-ever push.
		if synced.LastModifiedRemote != nil {
			expectedRemoteVersion = synced.LastModifiedRemote.Format(time.RFC3339)
		}
	}

	externalID, pushErr := adapter.Push(ctx, credentials, fm.ExternalContainerID, existingExternalID, properties, expectedRemoteVersion)
	if pushErr != nil {
		var conflict *integration.ConflictError
		if errors.As(pushErr, &conflict) {
			return s.markConflict(ctx, orgID, entityType, entityID, fm, synced, notFoundLocally)
		}
		if !notFoundLocally {
			s.recordFailure(ctx, synced, pushErr)
		}
		return fmt.Errorf("push %s %s to %s: %w", entityType, entityID, adapter.System(), pushErr)
	}

	now := time.Now()
	if notFoundLocally {
		synced = &models.SyncedEntity{
			OrganizationID:    orgID,
			IntegrationID:     integrationRec.ID,
			EntityType:        entityType,
			EntityID:          entityID,
			ExternalSystem:    adapter.System(),
			ExternalID:        externalID,
			SyncState:         "synced",
			Version:           1,
			Source:            "timeless",
			LastModifiedLocal: &now,
			LastSyncedAt:      &now,
			FieldMappingID:    &fm.ID,
		}
		if err := s.syncedRepo.Create(ctx, synced); err != nil {
			return fmt.Errorf("create sync ledger row: %w", err)
		}
	} else {
		synced.ExternalID = externalID
		synced.SyncState = "synced"
		synced.Version++
		synced.Source = "timeless"
		synced.LastModifiedLocal = &now
		synced.LastSyncedAt = &now
		synced.LastError = nil
		synced.ConflictState = ""
		if err := s.syncedRepo.Update(ctx, synced); err != nil {
			return fmt.Errorf("update sync ledger row: %w", err)
		}
	}
	s.recordHistory(ctx, synced, "pushed_to_remote", nil)
	return nil
}

func (s *PushService) markConflict(ctx context.Context, orgID uuid.UUID, entityType string, entityID uuid.UUID, fm *models.FieldMapping, synced *models.SyncedEntity, notFoundLocally bool) error {
	now := time.Now()
	if notFoundLocally {
		synced = &models.SyncedEntity{
			OrganizationID:    orgID,
			IntegrationID:     fm.IntegrationID,
			EntityType:        entityType,
			EntityID:          entityID,
			ExternalSystem:    "notion",
			SyncState:         "conflict",
			ConflictState:     "both_changed",
			LastModifiedLocal: &now,
			FieldMappingID:    &fm.ID,
		}
		return s.syncedRepo.Create(ctx, synced)
	}
	synced.SyncState = "conflict"
	synced.ConflictState = "both_changed"
	synced.LastModifiedLocal = &now
	if err := s.syncedRepo.Update(ctx, synced); err != nil {
		return err
	}
	s.recordHistory(ctx, synced, "conflict_detected", nil)
	return nil
}

func (s *PushService) recordFailure(ctx context.Context, synced *models.SyncedEntity, pushErr error) {
	errStr := pushErr.Error()
	synced.SyncState = "error"
	synced.LastError = &errStr
	_ = s.syncedRepo.Update(ctx, synced)
	s.recordHistory(ctx, synced, "sync_failed", pushErr)
}

func (s *PushService) recordHistory(ctx context.Context, synced *models.SyncedEntity, action string, historyErr error) {
	if synced == nil || synced.ID == uuid.Nil {
		return
	}
	h := &models.SyncHistory{
		SyncedEntityID: synced.ID,
		OrganizationID: synced.OrganizationID,
		Action:         action,
		Source:         "timeless",
	}
	if historyErr != nil {
		msg := historyErr.Error()
		h.Error = &msg
	}
	_ = s.historyRepo.Record(ctx, h)
}
