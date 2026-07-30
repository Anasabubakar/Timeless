package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"github.com/timeless/backend/internal/eventbus"
)

// TaskEventDispatch is the one asynq task type every eventbus.Event
// rides on for durable delivery — a single task type rather than one
// per event name, since the event's own Type field (carried in
// TaskPayload.Action) is what actually routes to subscribers via
// Bus.Dispatch; asynq only needs to know "there's an event to
// dispatch," not which one.
const TaskEventDispatch = "event:dispatch"

// NewEventPublisher builds an eventbus.Publisher that durably enqueues
// via the given worker client — the wiring router.Setup uses so
// eventbus.Bus.Publish survives a process restart instead of only ever
// dispatching in-process.
func NewEventPublisher(client *Client) eventbus.Publisher {
	return func(ctx context.Context, evt eventbus.Event) error {
		_, err := client.Enqueue(TaskEventDispatch, TaskPayload{
			OrgID:      evt.OrgID,
			EntityID:   evt.EntityID,
			EntityType: evt.EntityType,
			Action:     evt.Type,
			Data:       evt.Data,
		})
		if err != nil {
			return fmt.Errorf("enqueue event %s: %w", evt.Type, err)
		}
		return nil
	}
}

// HandleEventDispatch is the asynq handler side: pulls the event back
// out of the task payload and hands it to Bus.Dispatch, which fans it
// out to every subscriber registered for that event type. A subscriber
// error here fails the whole task, which asynq then retries per its
// normal retry policy — and, if retries are exhausted, dead-letters via
// the ErrorHandler wired in cmd/worker/main.go, same as any other task.
func (h *Handlers) HandleEventDispatch(ctx context.Context, t *asynq.Task) error {
	p, err := h.parsePayload(t)
	if err != nil {
		return err
	}
	if h.bus == nil {
		return nil // no bus configured (e.g. a stripped-down worker build) — nothing to do
	}
	evt := eventbus.Event{
		Type:       p.Action,
		OrgID:      p.OrgID,
		EntityType: p.EntityType,
		EntityID:   p.EntityID,
		Data:       p.Data,
		OccurredAt: time.Now(),
	}
	return h.bus.Dispatch(ctx, evt)
}
