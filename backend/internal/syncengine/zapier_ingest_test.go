package syncengine

import (
	"context"
	"testing"

	"github.com/timeless/backend/internal/eventbus"
)

// These exercise HandleEvent's early-return guards, which run before any
// repository/DB access — the only part of ZapierIngestService testable
// without a live database. A nil-dependency service is deliberately used
// to prove these paths never reach the repos at all; if they did, the
// test would panic on the nil pointer instead of returning cleanly.

func TestZapierIngestIgnoresUnrecognizedEventType(t *testing.T) {
	s := NewZapierIngestService(nil, nil, nil)
	err := s.HandleEvent(context.Background(), eventbus.Event{
		Type:  eventbus.ZapierWebhookReceived,
		OrgID: "11111111-1111-1111-1111-111111111111",
		Data:  map[string]interface{}{"event_type": "some_other_zap_action"},
	})
	if err != nil {
		t.Fatalf("expected a no-op for an unrecognized event_type, got error: %v", err)
	}
}

func TestZapierIngestIgnoresMissingEventType(t *testing.T) {
	s := NewZapierIngestService(nil, nil, nil)
	err := s.HandleEvent(context.Background(), eventbus.Event{
		Type:  eventbus.ZapierWebhookReceived,
		OrgID: "11111111-1111-1111-1111-111111111111",
		Data:  map[string]interface{}{"email": "a@example.com"},
	})
	if err != nil {
		t.Fatalf("expected a no-op when event_type is absent, got error: %v", err)
	}
}

func TestZapierIngestIgnoresContactPayloadWithNoIdentifyingFields(t *testing.T) {
	s := NewZapierIngestService(nil, nil, nil)
	err := s.HandleEvent(context.Background(), eventbus.Event{
		Type:  eventbus.ZapierWebhookReceived,
		OrgID: "11111111-1111-1111-1111-111111111111",
		Data:  map[string]interface{}{"event_type": "contact", "notes": "no name or email here"},
	})
	if err != nil {
		t.Fatalf("expected a no-op when there's nothing to identify a contact by, got error: %v", err)
	}
}

func TestZapierIngestRejectsInvalidOrgID(t *testing.T) {
	s := NewZapierIngestService(nil, nil, nil)
	err := s.HandleEvent(context.Background(), eventbus.Event{
		Type:  eventbus.ZapierWebhookReceived,
		OrgID: "not-a-uuid",
		Data:  map[string]interface{}{"event_type": "lead", "email": "a@example.com"},
	})
	if err == nil {
		t.Fatal("expected an error for an invalid org id")
	}
}

func TestStringFieldMissingKey(t *testing.T) {
	if got := stringField(map[string]interface{}{"a": "b"}, "missing"); got != "" {
		t.Errorf("got %q, want empty string for a missing key", got)
	}
}

func TestStringFieldWrongType(t *testing.T) {
	if got := stringField(map[string]interface{}{"n": 42}, "n"); got != "" {
		t.Errorf("got %q, want empty string for a non-string value", got)
	}
}
