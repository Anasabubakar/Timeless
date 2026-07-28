package worker

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

// resyncInterval is how stale an active integration's last_sync_at must be
// before the scheduler enqueues another sync. Real-time-ish for a
// background poll without hammering rate-limited providers.
const resyncInterval = 15 * time.Minute

const schedulerTick = 5 * time.Minute

// RecoverStaleSyncs runs once at worker startup: it reaps any sync_run left
// "running" by a crashed/killed previous process, then re-enqueues a fresh
// sync for every integration that was stuck mid-sync when that happened —
// the concrete implementation of "recover automatically after failures"
// rather than requiring a human to notice and manually reconnect.
func RecoverStaleSyncs(db *gorm.DB, syncRunRepo *repository.SyncRunRepository, client *Client, logger *slog.Logger) {
	ctx := context.Background()

	reaped, err := syncRunRepo.ReapStaleRuns(ctx)
	if err != nil {
		logger.Error("failed to reap stale sync runs", "error", err)
		return
	}
	if reaped == 0 {
		return
	}
	logger.Warn("reaped stale sync runs from a previous worker instance", "count", reaped)

	var stuck []models.Integration
	if err := db.Where("status IN ?", []string{"syncing", "retrying"}).Find(&stuck).Error; err != nil {
		logger.Error("failed to find stuck integrations after reaping", "error", err)
		return
	}

	for _, in := range stuck {
		running, err := syncRunRepo.HasRunning(ctx, in.ID)
		if err != nil || running {
			continue
		}
		if _, err := client.Enqueue(TaskIntegrationSync, TaskPayload{
			OrgID:      in.OrganizationID.String(),
			EntityID:   in.ID.String(),
			EntityType: "integration",
			Action:     in.Provider,
			Data:       map[string]interface{}{"trigger": "retry"},
		}); err != nil {
			logger.Error("failed to re-enqueue stuck integration", "integration_id", in.ID, "error", err)
			continue
		}
		logger.Info("re-enqueued sync for integration stuck by a previous worker crash", "integration_id", in.ID, "provider", in.Provider)
	}
}

// StartPeriodicResync launches the background scheduler that keeps every
// connected integration fresh even when nothing triggers a webhook or
// manual reconnect — this is the "efficient polling with change detection"
// fallback for providers/events that don't push a webhook. Returns a stop
// function to call on graceful shutdown.
func StartPeriodicResync(db *gorm.DB, syncRunRepo *repository.SyncRunRepository, cfg *config.Config, logger *slog.Logger) func() {
	client, err := NewClient(cfg)
	if err != nil {
		logger.Error("periodic resync: could not create enqueue client", "error", err)
		return func() {}
	}

	RecoverStaleSyncs(db, syncRunRepo, client, logger)

	ticker := time.NewTicker(schedulerTick)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				resyncDueIntegrations(db, syncRunRepo, client, logger)
			case <-done:
				return
			}
		}
	}()

	logger.Info("periodic integration resync scheduler started", "interval", resyncInterval.String())

	return func() {
		ticker.Stop()
		close(done)
		client.Close()
	}
}

func resyncDueIntegrations(db *gorm.DB, syncRunRepo *repository.SyncRunRepository, client *Client, logger *slog.Logger) {
	var integrations []models.Integration
	cutoff := time.Now().Add(-resyncInterval)
	if err := db.Where("status = ? AND (last_sync_at IS NULL OR last_sync_at < ?)", "active", cutoff).
		Find(&integrations).Error; err != nil {
		logger.Error("periodic resync: list integrations", "error", err)
		return
	}

	ctx := context.Background()
	for _, in := range integrations {
		running, err := syncRunRepo.HasRunning(ctx, in.ID)
		if err != nil || running {
			continue
		}
		if _, err := client.Enqueue(TaskIntegrationSync, TaskPayload{
			OrgID:      in.OrganizationID.String(),
			EntityID:   in.ID.String(),
			EntityType: "integration",
			Action:     in.Provider,
			Data:       map[string]interface{}{"trigger": "scheduled"},
		}); err != nil {
			logger.Error("periodic resync: enqueue failed", "integration_id", in.ID, "error", err)
			continue
		}
		logger.Info("periodic resync enqueued", "integration_id", in.ID, "provider", in.Provider)
	}
}
