package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

// NewDeadLetterHandler builds an asynq.ErrorHandlerFunc that fires on
// every task failure and specifically flags the ones that just
// exhausted their retry budget — asynq's own behavior at that point is
// to move the task to its "archived" set (its dead-letter queue,
// functionally: kept for inspection instead of retried or discarded),
// but nothing was watching for that happening. Without this, a
// permanently-failing task (a broken integration, a bad payload) would
// silently pile up in the archived set with no operator ever notified.
func NewDeadLetterHandler(db *gorm.DB, logger *slog.Logger) asynq.ErrorHandlerFunc {
	return func(ctx context.Context, task *asynq.Task, err error) {
		taskID, _ := asynq.GetTaskID(ctx)
		retryCount, okRetry := asynq.GetRetryCount(ctx)
		maxRetry, okMax := asynq.GetMaxRetry(ctx)

		isFinal := isLastAttempt(retryCount, maxRetry, okRetry, okMax)

		logger.Error("task failed",
			"task_type", task.Type(),
			"task_id", taskID,
			"retry_count", retryCount,
			"max_retry", maxRetry,
			"dead_lettered", isFinal,
			"error", err,
		)

		if !isFinal {
			return
		}

		logger.Error("task exhausted retries and was dead-lettered — needs operator attention",
			"task_type", task.Type(), "task_id", taskID)

		if db == nil {
			return
		}
		meta := map[string]string{
			"task_type":   task.Type(),
			"task_id":     taskID,
			"retry_count": strconv.Itoa(retryCount),
			"error":       err.Error(),
		}
		metaJSON, _ := json.Marshal(meta)
		activity := models.Activity{
			EntityType: "background_job",
			Type:       "task_dead_lettered",
			Subject:    "task exhausted retries: " + task.Type(),
			Metadata:   datatypes.JSON(metaJSON),
		}
		activity.ID = uuid.New()
		// Best-effort, synchronous (unlike the request-path
		// LogSecurityEvent helper): this already runs on a background
		// worker goroutine, not in the hot path of a user-facing request,
		// so there's no latency concern to dodge by going async.
		if err := db.Create(&activity).Error; err != nil {
			logger.Error("failed to record dead-letter event", "error", err)
		}
	}
}

// isLastAttempt reports whether a failure at retryCount (out of
// maxRetry) is the one that dead-letters the task — asynq retries while
// retryCount < maxRetry, so once a failure makes retryCount == maxRetry
// there's no next attempt. Both context values missing (okRetry/okMax
// false, which shouldn't happen inside a real asynq processor — only in
// a test or a future asynq version that changes this contract) is
// treated as "don't know, assume not final" rather than defaulting to
// the zero-value comparison 0 >= 0, which would misfire on every call.
func isLastAttempt(retryCount, maxRetry int, okRetry, okMax bool) bool {
	if !okRetry || !okMax {
		return false
	}
	return retryCount >= maxRetry
}
