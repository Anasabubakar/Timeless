package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

type SyncRunRepository struct {
	db *gorm.DB
}

func NewSyncRunRepository(db *gorm.DB) *SyncRunRepository {
	return &SyncRunRepository{db: db}
}

// Start records a new in-progress sync run so the dashboard can show
// "Syncing..." the moment a job picks up work, not just after it finishes.
func (r *SyncRunRepository) Start(ctx context.Context, orgID, integrationID uuid.UUID, provider, trigger string, attempt int) (*models.SyncRun, error) {
	run := &models.SyncRun{
		OrganizationID: orgID,
		IntegrationID:  integrationID,
		Provider:       provider,
		Trigger:        trigger,
		Status:         "running",
		StartedAt:      time.Now(),
		Attempt:        attempt,
	}
	if err := r.db.WithContext(ctx).Create(run).Error; err != nil {
		return nil, err
	}
	return run, nil
}

func (r *SyncRunRepository) Finish(ctx context.Context, runID uuid.UUID, status string, recordsSynced int, warnings []string, syncErr error, details map[string]interface{}) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":         status,
		"finished_at":    now,
		"records_synced": recordsSynced,
	}

	var run models.SyncRun
	if err := r.db.WithContext(ctx).Select("started_at").First(&run, "id = ?", runID).Error; err == nil {
		durationMS := now.Sub(run.StartedAt).Milliseconds()
		updates["duration_ms"] = durationMS
	}

	if syncErr != nil {
		errStr := syncErr.Error()
		updates["error"] = errStr
	}
	if len(warnings) > 0 {
		updates["warnings"] = mustJSON(warnings)
	}
	if details != nil {
		updates["details"] = mustJSON(details)
	}

	return r.db.WithContext(ctx).Model(&models.SyncRun{}).Where("id = ?", runID).Updates(updates).Error
}

func (r *SyncRunRepository) ListByIntegration(ctx context.Context, orgID, integrationID uuid.UUID, limit int) ([]models.SyncRun, error) {
	var runs []models.SyncRun
	q := r.db.WithContext(ctx).
		Where("organization_id = ? AND integration_id = ?", orgID, integrationID).
		Order("started_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&runs).Error
	return runs, err
}

func (r *SyncRunRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]models.SyncRun, error) {
	var runs []models.SyncRun
	q := r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Order("started_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&runs).Error
	return runs, err
}

// staleRunThreshold bounds how long a "running" sync_run is trusted before
// it's treated as abandoned (worker crashed/restarted mid-sync) rather than
// genuinely still in flight. Without this, a single killed worker process
// would permanently wedge HasRunning() to true for that integration —
// nothing would ever sync again.
const staleRunThreshold = 10 * time.Minute

// HasRunning checks whether a sync is already in flight for this
// integration, so the scheduler and webhook triggers never enqueue a
// duplicate concurrent job for the same connection. A "running" row older
// than staleRunThreshold doesn't count — see ReapStaleRuns.
func (r *SyncRunRepository) HasRunning(ctx context.Context, integrationID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.SyncRun{}).
		Where("integration_id = ? AND status = ? AND started_at > ?", integrationID, "running", time.Now().Add(-staleRunThreshold)).
		Count(&count).Error
	return count > 0, err
}

// ReapStaleRuns marks any "running" sync_run older than staleRunThreshold
// as failed — recovery for a worker process that crashed or was
// force-killed mid-sync, so those integrations aren't wedged forever.
// Returns the number of runs reaped.
func (r *SyncRunRepository) ReapStaleRuns(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.SyncRun{}).
		Where("status = ? AND started_at <= ?", "running", time.Now().Add(-staleRunThreshold)).
		Updates(map[string]interface{}{
			"status": "failed",
			"error":  "worker process stopped before this sync finished (reaped on restart)",
		})
	return result.RowsAffected, result.Error
}
