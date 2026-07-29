package handler

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/pkg/reqbind"
)

// TestCompanyInputRejectsMassAssignment is the regression test for the
// bug class this whole phase has been fixing: binding a request body
// directly into a GORM model lets a client set fields (id,
// organization_id, created_at) it should never control. CompanyInput +
// reqbind.JSON should reject a body that tries.
func TestCompanyInputRejectsMassAssignment(t *testing.T) {
	forgedID := uuid.New().String()
	forgedOrgID := uuid.New().String()
	body := `{
		"name": "Acme Inc",
		"id": "` + forgedID + `",
		"organization_id": "` + forgedOrgID + `",
		"created_at": "2000-01-01T00:00:00Z"
	}`

	var input CompanyInput
	verr := reqbind.JSONFromBytes([]byte(body), &input)
	if verr == nil {
		t.Fatal("expected reqbind to reject id/organization_id/created_at as unknown fields, got no error")
	}
}

// TestCompanyInputApplyToOnlyTouchesDeclaredFields confirms applyTo
// can't be used to smuggle in an ID/org change even if a caller
// constructed CompanyInput directly (bypassing JSON entirely) — the
// struct itself has no field for it.
func TestCompanyInputApplyToOnlyTouchesDeclaredFields(t *testing.T) {
	orgID := uuid.New()
	id := uuid.New()
	company := &models.Company{OrganizationID: orgID}
	company.ID = id

	input := CompanyInput{Name: "Acme Inc"}
	input.applyTo(company)

	if company.OrganizationID != orgID {
		t.Errorf("applyTo must not change OrganizationID, got %v want %v", company.OrganizationID, orgID)
	}
	if company.ID != id {
		t.Errorf("applyTo must not change ID, got %v want %v", company.ID, id)
	}
	if company.Name != "Acme Inc" {
		t.Errorf("applyTo should set Name, got %q", company.Name)
	}
}

func TestCompanyInputRequiresName(t *testing.T) {
	body := `{"domain": "acme.com"}`
	var input CompanyInput
	verr := reqbind.JSONFromBytes([]byte(body), &input)
	if verr == nil {
		t.Fatal("expected a validation error for missing required name")
	}
	if !strings.Contains(verr.Message, "Validation") && verr.Details == nil {
		t.Errorf("expected validation details, got %+v", verr)
	}
}
