// Package eventbus is the central event-driven backbone: every entity
// change (created/updated/completed) and every integration signal
// (Notion changed, Zapier webhook received, AI finished research)
// produces an Event; subscribers (currently the sync pipeline; future
// integrations) register interest in specific event types instead of
// being called directly by whatever produced the event. Durability comes
// from routing Publish through asynq (see the SetPublisher wiring in
// router.Setup and the worker.TaskEventDispatch handler) — an event
// survives a process restart the same way any other background job does.
package eventbus

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Event type constants — the vocabulary every producer/subscriber in
// this codebase shares. Adding a new one is additive (new subscribers
// opt in), never a breaking change to existing publishers.
const (
	CompanyCreated  = "CompanyCreated"
	CompanyUpdated  = "CompanyUpdated"
	CompanyDeleted  = "CompanyDeleted"
	ContactCreated  = "ContactCreated"
	ContactUpdated  = "ContactUpdated"
	ContactDeleted  = "ContactDeleted"
	SponsorCreated  = "SponsorCreated"
	SponsorUpdated  = "SponsorUpdated"
	SponsorDeleted  = "SponsorDeleted"
	MeetingScheduled = "MeetingScheduled"
	MeetingUpdated  = "MeetingUpdated"
	TaskCreated     = "TaskCreated"
	TaskCompleted   = "TaskCompleted"
	TaskUpdated     = "TaskUpdated"
	ProjectUpdated  = "ProjectUpdated"
	NoteCreated     = "NoteCreated"
	NoteUpdated     = "NoteUpdated"

	NotionChanged  = "NotionChanged"
	ApolloUpdated  = "ApolloUpdated"
	ZapierWebhookReceived = "ZapierWebhookReceived"
	EmailReceived  = "EmailReceived"
	CalendarChanged = "CalendarChanged"

	AICompletedResearch = "AICompletedResearch"
	SponsorQualified    = "SponsorQualified"
)

// Event is the one shape every producer emits and every subscriber
// receives, regardless of what triggered it.
type Event struct {
	Type       string                 `json:"type"`
	OrgID      string                 `json:"org_id"`
	EntityType string                 `json:"entity_type,omitempty"`
	EntityID   string                 `json:"entity_id,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
	OccurredAt time.Time              `json:"occurred_at"`
}

// Handler processes one event. A subscriber returning an error causes
// the whole dispatch (all subscribers for this event, via the asynq
// task wrapping it) to be retried — see worker.TaskEventDispatch.
type Handler func(ctx context.Context, evt Event) error

// Publisher durably delivers an event — the production implementation
// enqueues it as an asynq task (see router.Setup) so it survives a
// process restart and gets asynq's retry/dead-letter handling for free;
// tests can substitute a synchronous stub.
type Publisher func(ctx context.Context, evt Event) error

// Bus is the central pub/sub registry. Safe for concurrent use.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[string][]Handler
	publish     Publisher
}

func NewBus() *Bus {
	return &Bus{subscribers: make(map[string][]Handler)}
}

// SetPublisher wires the durable delivery path. Until this is called,
// Publish dispatches synchronously in-process (fine for tests; not
// durable — a crash between publish and subscriber completion loses the
// event, which is exactly what routing through asynq avoids).
func (b *Bus) SetPublisher(p Publisher) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.publish = p
}

// Subscribe registers a handler for an event type. Multiple handlers
// per type are expected — that's the point of a bus over direct calls.
func (b *Bus) Subscribe(eventType string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[eventType] = append(b.subscribers[eventType], h)
}

// Publish emits an event for durable delivery (via the configured
// Publisher) if one is set, otherwise dispatches synchronously.
func (b *Bus) Publish(ctx context.Context, evt Event) error {
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now()
	}
	b.mu.RLock()
	publish := b.publish
	b.mu.RUnlock()

	if publish != nil {
		return publish(ctx, evt)
	}
	return b.Dispatch(ctx, evt)
}

// Dispatch invokes every subscriber registered for evt.Type. Called
// directly by Publish when no durable Publisher is configured, and by
// the asynq event-dispatch task handler on the worker side once an
// event comes back off the queue — same fan-out logic either way.
func (b *Bus) Dispatch(ctx context.Context, evt Event) error {
	b.mu.RLock()
	handlers := append([]Handler(nil), b.subscribers[evt.Type]...)
	b.mu.RUnlock()

	var errs []error
	for _, h := range handlers {
		if err := h(ctx, evt); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("eventbus: %d of %d subscriber(s) for %s failed: %w", len(errs), len(handlers), evt.Type, errs[0])
	}
	return nil
}
