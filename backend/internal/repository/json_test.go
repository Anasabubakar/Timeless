package repository

import (
	"encoding/json"
	"testing"
)

func TestMustJSONMarshalsValidValues(t *testing.T) {
	raw := mustJSON([]string{"a", "b"})
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("mustJSON output did not round-trip: %v", err)
	}
	if len(out) != 2 || out[0] != "a" || out[1] != "b" {
		t.Errorf("mustJSON round-trip = %v, want [a b]", out)
	}
}

func TestMustJSONFallsBackToNullOnUnmarshalableValue(t *testing.T) {
	// Channels can't be marshaled to JSON — mustJSON must not panic.
	raw := mustJSON(make(chan int))
	if string(raw) != "null" {
		t.Errorf("mustJSON(unmarshalable) = %s, want null", raw)
	}
}
