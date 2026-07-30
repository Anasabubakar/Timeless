package mapping

import (
	"encoding/json"
	"testing"

	"github.com/timeless/backend/internal/models"
)

func fieldMappingWith(entries []models.FieldMappingEntry) *models.FieldMapping {
	b, _ := json.Marshal(entries)
	return &models.FieldMapping{Fields: b}
}

func TestParseFieldsRoundTrips(t *testing.T) {
	entries := []models.FieldMappingEntry{
		{InternalField: "name", ExternalField: "Name", ExternalType: "title", Direction: "both"},
		{InternalField: "status", ExternalField: "Status", ExternalType: "status", Direction: "to_external"},
	}
	fm := fieldMappingWith(entries)

	got, err := ParseFields(fm)
	if err != nil {
		t.Fatalf("ParseFields: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if got[0].InternalField != "name" || got[1].ExternalType != "status" {
		t.Errorf("unexpected entries: %+v", got)
	}
}

func TestParseFieldsNilMapping(t *testing.T) {
	got, err := ParseFields(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for a nil mapping, got %v", got)
	}
}

func TestParseFieldsRejectsInvalidJSON(t *testing.T) {
	fm := &models.FieldMapping{Fields: []byte("not json")}
	if _, err := ParseFields(fm); err == nil {
		t.Error("expected an error for malformed Fields JSON")
	}
}

func TestEntriesForDirectionDefaultsEmptyToBoth(t *testing.T) {
	entries := []models.FieldMappingEntry{
		{InternalField: "a", Direction: ""},
		{InternalField: "b", Direction: "both"},
		{InternalField: "c", Direction: "to_external"},
		{InternalField: "d", Direction: "from_external"},
	}

	toExternal := entriesForDirection(entries, "to_external")
	if len(toExternal) != 3 { // a, b, c
		t.Errorf("to_external: got %d entries, want 3: %+v", len(toExternal), toExternal)
	}

	fromExternal := entriesForDirection(entries, "from_external")
	if len(fromExternal) != 3 { // a, b, d
		t.Errorf("from_external: got %d entries, want 3: %+v", len(fromExternal), fromExternal)
	}
}
