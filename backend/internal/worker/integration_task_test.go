package worker

import (
	"encoding/json"
	"testing"

	"gorm.io/datatypes"
)

func TestExtractStateRoundTrip(t *testing.T) {
	config := mergeConfig(datatypes.JSON([]byte("{}")), map[string]interface{}{"pages_found": 3}, map[string]interface{}{"watermark": "2026-01-01T00:00:00Z"})

	state := extractState(config)
	if state["watermark"] != "2026-01-01T00:00:00Z" {
		t.Errorf("extractState() watermark = %v, want %q", state["watermark"], "2026-01-01T00:00:00Z")
	}
}

func TestMergeConfigPreservesPriorState(t *testing.T) {
	first := mergeConfig(datatypes.JSON([]byte("{}")), map[string]interface{}{"pages_found": 1}, map[string]interface{}{"watermark": "A"})
	second := mergeConfig(first, map[string]interface{}{"pages_found": 2}, map[string]interface{}{})

	state := extractState(second)
	if state["watermark"] != "A" {
		t.Errorf("expected watermark from the first run to survive a second run with no new state, got %v", state["watermark"])
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(second, &parsed); err != nil {
		t.Fatalf("mergeConfig produced invalid JSON: %v", err)
	}
	details, _ := parsed["details"].(map[string]interface{})
	if details["pages_found"] != float64(2) {
		t.Errorf("expected details to be replaced by the latest run, got %v", details["pages_found"])
	}
}

func TestMergeConfigOverwritesWatermark(t *testing.T) {
	first := mergeConfig(datatypes.JSON([]byte("{}")), nil, map[string]interface{}{"watermark": "A"})
	second := mergeConfig(first, nil, map[string]interface{}{"watermark": "B"})

	state := extractState(second)
	if state["watermark"] != "B" {
		t.Errorf("expected the newer watermark to win, got %v", state["watermark"])
	}
}

func TestSplitFullName(t *testing.T) {
	cases := []struct {
		in          string
		first, last string
	}{
		{"", "", ""},
		{"Cher", "Cher", ""},
		{"Jane Doe", "Jane", "Doe"},
		{"Mary Jane Watson", "Mary Jane", "Watson"},
	}
	for _, c := range cases {
		first, last := splitFullName(c.in)
		if first != c.first || last != c.last {
			t.Errorf("splitFullName(%q) = (%q, %q), want (%q, %q)", c.in, first, last, c.first, c.last)
		}
	}
}
