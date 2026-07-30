package mapping

import (
	"testing"

	"github.com/google/uuid"

	"github.com/timeless/backend/internal/models"
)

func strp(s string) *string { return &s }

func TestCompanyToRecordOmitsNilOptionalFields(t *testing.T) {
	id := uuid.New()
	c := &models.Company{
		Base:   models.Base{ID: id},
		Name:   "Acme Inc",
		Status: "active",
	}
	rec := CompanyToRecord(c)

	if rec.EntityType != "company" || rec.EntityID != id {
		t.Fatalf("unexpected record identity: %+v", rec)
	}
	if rec.Fields["name"] != "Acme Inc" {
		t.Errorf("name = %v", rec.Fields["name"])
	}
	if _, ok := rec.Fields["domain"]; ok {
		t.Error("a nil *string field should be omitted, not present as empty")
	}
}

func TestCompanyToRecordIncludesSetOptionalFields(t *testing.T) {
	c := &models.Company{
		Base:    models.Base{ID: uuid.New()},
		Name:    "Acme Inc",
		Status:  "active",
		Domain:  strp("acme.com"),
		Website: strp("https://acme.com"),
	}
	rec := CompanyToRecord(c)
	if rec.Fields["domain"] != "acme.com" {
		t.Errorf("domain = %v, want acme.com", rec.Fields["domain"])
	}
	if rec.Fields["website"] != "https://acme.com" {
		t.Errorf("website = %v", rec.Fields["website"])
	}
}

func TestContactToRecord(t *testing.T) {
	c := &models.Contact{
		Base:      models.Base{ID: uuid.New()},
		FirstName: "Ada",
		LastName:  "Lovelace",
		Status:    "active",
		Email:     strp("ada@example.com"),
	}
	rec := ContactToRecord(c)
	if rec.EntityType != "contact" {
		t.Errorf("entity type = %q, want contact", rec.EntityType)
	}
	if rec.Fields["first_name"] != "Ada" || rec.Fields["last_name"] != "Lovelace" {
		t.Errorf("unexpected name fields: %+v", rec.Fields)
	}
	if rec.Fields["email"] != "ada@example.com" {
		t.Errorf("email = %v", rec.Fields["email"])
	}
}

func TestSponsorToRecordIncludesCompanyNameWhenLoaded(t *testing.T) {
	s := &models.Sponsor{
		Base:  models.Base{ID: uuid.New()},
		Stage: "prospect",
		Company: models.Company{
			Name: "Acme Inc",
		},
	}
	rec := SponsorToRecord(s)
	if rec.Fields["stage"] != "prospect" {
		t.Errorf("stage = %v", rec.Fields["stage"])
	}
	if rec.Fields["company_name"] != "Acme Inc" {
		t.Errorf("company_name = %v, want Acme Inc", rec.Fields["company_name"])
	}
}

func TestSponsorToRecordOmitsCompanyNameWhenNotLoaded(t *testing.T) {
	s := &models.Sponsor{Base: models.Base{ID: uuid.New()}, Stage: "prospect"}
	rec := SponsorToRecord(s)
	if _, ok := rec.Fields["company_name"]; ok {
		t.Error("expected company_name to be omitted when Company wasn't preloaded")
	}
}
