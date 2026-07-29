package agent

import (
	"testing"

	"github.com/timeless/backend/internal/ai/provider"
)

func TestToolAllowlist(t *testing.T) {
	allowlist := NewToolAllowlist("search_companies", "get_contact")

	if !allowlist.IsAllowed("search_companies") {
		t.Error("expected search_companies to be allowed")
	}
	if allowlist.IsAllowed("delete_organization") {
		t.Error("expected delete_organization (not on the list) to be disallowed")
	}
}

func TestToolAllowlistValidateCall(t *testing.T) {
	allowlist := NewToolAllowlist("search_companies")

	cases := []struct {
		name    string
		call    provider.ToolCall
		wantErr bool
	}{
		{"allowed tool passes", provider.ToolCall{Name: "search_companies", Arguments: "{}"}, false},
		{"disallowed tool is rejected", provider.ToolCall{Name: "drop_database", Arguments: "{}"}, true},
		{"empty tool name is rejected", provider.ToolCall{Name: "", Arguments: "{}"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := allowlist.ValidateCall(tc.call)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateCall(%+v) error = %v, wantErr %v", tc.call, err, tc.wantErr)
			}
		})
	}
}

func TestNewToolAllowlistEmpty(t *testing.T) {
	allowlist := NewToolAllowlist()
	if allowlist.IsAllowed("anything") {
		t.Error("an empty allowlist must allow nothing")
	}
}
