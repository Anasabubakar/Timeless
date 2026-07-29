package worker

import (
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

// retentionTick is how often the retention sweep runs — daily is
// frequent enough for a purge window measured in days/months without
// adding meaningful load.
const retentionTick = 24 * time.Hour

// StartActivityRetention launches a background sweep that hard-deletes
// activity (audit log) rows older than retentionDays, if a retention
// window is configured at all. A 0/negative retentionDays disables
// purging entirely — audit history is kept forever until an operator
// explicitly opts into a retention window, since silently losing audit
// history is a worse default than an unbounded table. Returns a stop
// function for graceful shutdown, matching StartPeriodicResync.
func StartActivityRetention(db *gorm.DB, retentionDays int, logger *slog.Logger) func() {
	if retentionDays <= 0 {
		logger.Info("activity retention disabled (AUDIT_LOG_RETENTION_DAYS not set) — audit log kept indefinitely")
		return func() {}
	}

	ticker := time.NewTicker(retentionTick)
	done := make(chan struct{})

	purgeOldActivities(db, retentionDays, logger) // run once at startup, don't wait a full tick

	go func() {
		for {
			select {
			case <-ticker.C:
				purgeOldActivities(db, retentionDays, logger)
			case <-done:
				return
			}
		}
	}()

	logger.Info("activity retention sweep started", "retention_days", retentionDays, "interval", retentionTick.String())

	return func() {
		ticker.Stop()
		close(done)
	}
}

func purgeOldActivities(db *gorm.DB, retentionDays int, logger *slog.Logger) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	// Activity rows are immutable (see enforceActivityImmutability) but
	// not un-deletable — Unscoped() issues a real DELETE rather than a
	// soft-delete UPDATE, which the immutability trigger would reject
	// anyway. This is the one sanctioned way old audit rows go away.
	result := db.Unscoped().Where("created_at < ?", cutoff).Delete(&models.Activity{})
	if result.Error != nil {
		logger.Error("activity retention sweep failed", "error", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		logger.Info("activity retention sweep purged old audit rows", "count", result.RowsAffected, "cutoff", cutoff.Format(time.RFC3339))
	}
}
