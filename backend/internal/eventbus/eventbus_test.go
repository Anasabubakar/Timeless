package eventbus

import (
	"context"
	"errors"
	"testing"
)

func TestDispatchFansOutToAllSubscribers(t *testing.T) {
	b := NewBus()
	var calls []string
	b.Subscribe(CompanyCreated, func(ctx context.Context, evt Event) error {
		calls = append(calls, "first")
		return nil
	})
	b.Subscribe(CompanyCreated, func(ctx context.Context, evt Event) error {
		calls = append(calls, "second")
		return nil
	})

	if err := b.Dispatch(context.Background(), Event{Type: CompanyCreated}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 subscriber calls, got %d", len(calls))
	}
}

func TestDispatchOnlyInvokesMatchingType(t *testing.T) {
	b := NewBus()
	invoked := false
	b.Subscribe(ContactCreated, func(ctx context.Context, evt Event) error {
		invoked = true
		return nil
	})

	if err := b.Dispatch(context.Background(), Event{Type: CompanyCreated}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invoked {
		t.Fatal("subscriber for a different event type should not be invoked")
	}
}

func TestDispatchAggregatesSubscriberErrors(t *testing.T) {
	b := NewBus()
	b.Subscribe(TaskCreated, func(ctx context.Context, evt Event) error {
		return errors.New("boom")
	})
	b.Subscribe(TaskCreated, func(ctx context.Context, evt Event) error {
		return nil
	})

	err := b.Dispatch(context.Background(), Event{Type: TaskCreated})
	if err == nil {
		t.Fatal("expected an aggregated error when a subscriber fails")
	}
}

func TestPublishUsesDurablePublisherWhenSet(t *testing.T) {
	b := NewBus()
	var published Event
	b.SetPublisher(func(ctx context.Context, evt Event) error {
		published = evt
		return nil
	})
	dispatched := false
	b.Subscribe(NoteCreated, func(ctx context.Context, evt Event) error {
		dispatched = true
		return nil
	})

	if err := b.Publish(context.Background(), Event{Type: NoteCreated}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if published.Type != NoteCreated {
		t.Fatal("expected the durable publisher to receive the event")
	}
	if dispatched {
		t.Fatal("Publish should route through the durable publisher, not dispatch synchronously, once one is set")
	}
}

func TestPublishDispatchesSynchronouslyWithoutPublisher(t *testing.T) {
	b := NewBus()
	dispatched := false
	b.Subscribe(MeetingScheduled, func(ctx context.Context, evt Event) error {
		dispatched = true
		return nil
	})

	if err := b.Publish(context.Background(), Event{Type: MeetingScheduled}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dispatched {
		t.Fatal("expected synchronous dispatch when no durable publisher is configured")
	}
}

func TestPublishStampsOccurredAtWhenZero(t *testing.T) {
	b := NewBus()
	var got Event
	b.Subscribe(SponsorCreated, func(ctx context.Context, evt Event) error {
		got = evt
		return nil
	})
	if err := b.Publish(context.Background(), Event{Type: SponsorCreated}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.OccurredAt.IsZero() {
		t.Fatal("expected OccurredAt to be stamped when left zero")
	}
}
