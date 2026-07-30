package handler

import (
	"testing"

	"github.com/timeless/backend/internal/pkg/reqbind"
)

func TestConnectInputRejectsEmptyCredentials(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"missing credentials field", `{}`},
		{"empty credentials map", `{"credentials": {}}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var input connectInput
			if verr := reqbind.JSONFromBytes([]byte(tc.body), &input); verr == nil {
				t.Fatal("expected empty/missing credentials to be rejected")
			}
		})
	}
}

func TestConnectInputAcceptsCredentials(t *testing.T) {
	body := `{"credentials": {"token": "some-token-value"}}`
	var input connectInput
	if verr := reqbind.JSONFromBytes([]byte(body), &input); verr != nil {
		t.Fatalf("expected valid credentials to be accepted, got %v", verr)
	}
	if input.Credentials["token"] != "some-token-value" {
		t.Errorf("credentials not populated correctly: %+v", input.Credentials)
	}
}

func TestConnectInputRejectsUnknownFields(t *testing.T) {
	body := `{"credentials": {"token": "x"}, "organization_id": "should-not-be-settable"}`
	var input connectInput
	if verr := reqbind.JSONFromBytes([]byte(body), &input); verr == nil {
		t.Fatal("expected an unknown top-level field to be rejected")
	}
}
