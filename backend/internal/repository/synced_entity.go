package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

type SyncedEntityRepository struct {
	db *gorm.DB
}

func NewSyncedEntityRepository(db *gorm.DB) *SyncedEntityRepository {
	return &SyncedEntityRepository{db: db}
}

func (r *SyncedEntityRepository) Create(ctx context.Context, e *models.SyncedEntity) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *SyncedEntityRepository) Update(ctx context.Context, e *models.SyncedEntity) error {
	return r.db.WithContext(ctx).Save(e).Error
}

// FindByInternal looks up the sync ledger row for one internal
// entity+external system pair — the common case ("did we already sync
// this company to Notion?").
func (r *SyncedEntityRepository) FindByInternal(ctx context.Context, orgID uuid.UUID, entityType string, entityID uuid.UUID, externalSystem string) (*models.SyncedEntity, error) {
	var e models.SyncedEntity
	err := r.db.WithContext(ctx).First(&e,
		"organization_id = ? AND entity_type = ? AND entity_id = ? AND external_system = ?",
		orgID, entityType, entityID, externalSystem,
	).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// FindByExternal is the inbound-webhook path: "a Notion page changed,
// which internal record does it map to?"
func (r *SyncedEntityRepository) FindByExternal(ctx context.Context, orgID uuid.UUID, externalSystem, externalID string) (*models.SyncedEntity, error) {
	var e models.SyncedEntity
	err := r.db.WithContext(ctx).First(&e,
		"organization_id = ? AND external_system = ? AND external_id = ?",
		orgID, externalSystem, externalID,
	).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *SyncedEntityRepository) ListByIntegration(ctx context.Context, orgID, integrationID uuid.UUID, limit, offset int) ([]models.SyncedEntity, int64, error) {
	var entities []models.SyncedEntity
	var total int64
	q := r.db.WithContext(ctx).Where("organization_id = ? AND integration_id = ?", orgID, integrationID)
	q.Model(&models.SyncedEntity{}).Count(&total)
	err := q.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&entities).Error
	return entities, total, err
}

// ListConflicts returns every SyncedEntity currently awaiting conflict
// resolution for an org — the Integration Dashboard's "conflict queue".
func (r *SyncedEntityRepository) ListConflicts(ctx context.Context, orgID uuid.UUID) ([]models.SyncedEntity, error) {
	var entities []models.SyncedEntity
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND sync_state = ?", orgID, "conflict").
		Order("updated_at DESC").
		Find(&entities).Error
	return entities, err
}

// CountByState powers the dashboard's synced/pending/failed counters.
func (r *SyncedEntityRepository) CountByState(ctx context.Context, orgID, integrationID uuid.UUID) (map[string]int64, error) {
	type row struct {
		SyncState string
		Count     int64
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&models.SyncedEntity{}).
		Select("sync_state, count(*) as count").
		Where("organization_id = ? AND integration_id = ?", orgID, integrationID).
		Group("sync_state").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(rows))
	for _, r := range rows {
		counts[r.SyncState] = r.Count
	}
	return counts, nil
}

type SyncHistoryRepository struct {
	db *gorm.DB
}

func NewSyncHistoryRepository(db *gorm.DB) *SyncHistoryRepository {
	return &SyncHistoryRepository{db: db}
}

func (r *SyncHistoryRepository) Record(ctx context.Context, h *models.SyncHistory) error {
	return r.db.WithContext(ctx).Create(h).Error
}

func (r *SyncHistoryRepository) ListForEntity(ctx context.Context, syncedEntityID uuid.UUID, limit int) ([]models.SyncHistory, error) {
	var history []models.SyncHistory
	err := r.db.WithContext(ctx).
		Where("synced_entity_id = ?", syncedEntityID).
		Order("created_at DESC").
		Limit(limit).
		Find(&history).Error
	return history, err
}

// ListRecentForOrg is the Sync Dashboard's activity feed: every sync
// action (pushed/pulled/conflict detected/resolved/failed) across every
// entity and integration in the org, newest first.
func (r *SyncHistoryRepository) ListRecentForOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]models.SyncHistory, error) {
	var history []models.SyncHistory
	err := r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Limit(limit).
		Find(&history).Error
	return history, err
}
