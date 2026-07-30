package mapping

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/timeless/backend/internal/models"
)

func TestNotionAdapterToExternalOnlyAppliesToExternalDirection(t *testing.T) {
	a := NewNotionAdapter(nil)
	fm := fieldMappingWith([]models.FieldMappingEntry{
		{InternalField: "name", ExternalField: "Name", ExternalType: "title", Direction: "both"},
		{InternalField: "internal_note", ExternalField: "Note", ExternalType: "rich_text", Direction: "from_external"},
	})
	record := SyncableRecord{
		EntityType: "company",
		EntityID:   uuid.New(),
		Fields: map[string]interface{}{
			"name":          "Acme Inc",
			"internal_note": "should not be pushed",
		},
	}

	props, err := a.ToExternal(record, fm)
	if err != nil {
		t.Fatalf("ToExternal: %v", err)
	}
	if _, ok := props["Note"]; ok {
		t.Error("a from_external-only field should not appear in outbound properties")
	}
	titleProp, ok := props["Name"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected a title property, got %v", props["Name"])
	}
	texts, _ := titleProp["title"].([]map[string]interface{})
	if len(texts) != 1 || texts[0]["text"].(map[string]interface{})["content"] != "Acme Inc" {
		t.Errorf("unexpected title payload: %+v", titleProp)
	}
}

func TestNotionAdapterToExternalSkipsUnmappedAndNilFields(t *testing.T) {
	a := NewNotionAdapter(nil)
	fm := fieldMappingWith([]models.FieldMappingEntry{
		{InternalField: "name", ExternalField: "Name", ExternalType: "title"},
		{InternalField: "domain", ExternalField: "Domain", ExternalType: "url"},
	})
	record := SyncableRecord{
		Fields: map[string]interface{}{
			"name": "Acme Inc",
			// "domain" deliberately absent
		},
	}

	props, err := a.ToExternal(record, fm)
	if err != nil {
		t.Fatalf("ToExternal: %v", err)
	}
	if _, ok := props["Domain"]; ok {
		t.Error("a missing internal field should be skipped, not sent as null")
	}
	if len(props) != 1 {
		t.Errorf("expected exactly 1 property, got %d: %+v", len(props), props)
	}
}

func TestNotionAdapterFromExternalDeserializesKnownTypes(t *testing.T) {
	a := NewNotionAdapter(nil)
	fm := fieldMappingWith([]models.FieldMappingEntry{
		{InternalField: "name", ExternalField: "Name", ExternalType: "title"},
		{InternalField: "stage", ExternalField: "Stage", ExternalType: "select"},
		{InternalField: "active", ExternalField: "Active", ExternalType: "checkbox"},
	})
	raw := map[string]interface{}{
		"Name":   map[string]interface{}{"title": []interface{}{map[string]interface{}{"plain_text": "Acme"}}},
		"Stage":  map[string]interface{}{"select": map[string]interface{}{"name": "prospect"}},
		"Active": map[string]interface{}{"checkbox": true},
	}

	got, err := a.FromExternal(raw, fm)
	if err != nil {
		t.Fatalf("FromExternal: %v", err)
	}
	if got["name"] != "Acme" {
		t.Errorf("name = %v, want Acme", got["name"])
	}
	if got["stage"] != "prospect" {
		t.Errorf("stage = %v, want prospect", got["stage"])
	}
	if got["active"] != true {
		t.Errorf("active = %v, want true", got["active"])
	}
}

func TestSerializeNotionPropertyUnsupportedTypeErrors(t *testing.T) {
	if _, err := serializeNotionProperty("not_a_real_type", "x"); err == nil {
		t.Error("expected an error for an unsupported Notion property type")
	}
}

func TestSerializeDeserializeDateRoundTrip(t *testing.T) {
	when := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	prop, err := serializeNotionProperty("date", when)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	value, ok := deserializeNotionProperty("date", prop)
	if !ok {
		t.Fatal("expected deserialization to succeed")
	}
	if value != "2026-07-30" {
		t.Errorf("got %v, want 2026-07-30", value)
	}
}

// jsonRoundTrip mimics what actually happens between serialize and
// deserialize in production: the serialized property is sent to Notion as
// JSON, and whatever comes back is decoded by encoding/json — which
// produces []interface{} for arrays, not the []map[string]interface{}
// serializeNotionProperty builds Go-natively. Deserializing straight from
// serialize's own output (skipping this) isn't a real code path.
func jsonRoundTrip(t *testing.T, v map[string]interface{}) map[string]interface{} {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

func TestSerializeMultiSelectAndRelation(t *testing.T) {
	prop, err := serializeNotionProperty("multi_select", []string{"a", "b"})
	if err != nil {
		t.Fatalf("serialize multi_select: %v", err)
	}
	names, ok := deserializeNotionProperty("multi_select", jsonRoundTrip(t, prop))
	if !ok {
		t.Fatal("expected multi_select deserialization to succeed")
	}
	got := names.([]string)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("got %v, want [a b]", got)
	}

	relProp, err := serializeNotionProperty("relation", []string{"page-1", "page-2"})
	if err != nil {
		t.Fatalf("serialize relation: %v", err)
	}
	ids, ok := deserializeNotionProperty("relation", jsonRoundTrip(t, relProp))
	if !ok {
		t.Fatal("expected relation deserialization to succeed")
	}
	gotIDs := ids.([]string)
	if len(gotIDs) != 2 || gotIDs[0] != "page-1" {
		t.Errorf("got %v", gotIDs)
	}
}

func TestDeserializeEmptyPropertyReturnsNotOK(t *testing.T) {
	if _, ok := deserializeNotionProperty("select", map[string]interface{}{"select": nil}); ok {
		t.Error("expected ok=false for an empty select property")
	}
	if _, ok := deserializeNotionProperty("date", map[string]interface{}{"date": nil}); ok {
		t.Error("expected ok=false for an empty date property")
	}
}
