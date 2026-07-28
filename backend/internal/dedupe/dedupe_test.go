package dedupe

import (
	"testing"

	"github.com/timeless/backend/internal/models"
)

func strPtr(s string) *string { return &s }

func TestDedupeKeyPrefersDomain(t *testing.T) {
	company := models.Company{Name: "Acme Inc", Domain: strPtr("Acme.com")}
	if got := dedupeKey(company); got != "domain:acme.com" {
		t.Errorf("dedupeKey() = %q, want %q", got, "domain:acme.com")
	}
}

func TestDedupeKeyFallsBackToName(t *testing.T) {
	company := models.Company{Name: "Acme Inc."}
	if got := dedupeKey(company); got != "name:acme" {
		t.Errorf("dedupeKey() = %q, want %q", got, "name:acme")
	}
}

func TestCompletenessScore(t *testing.T) {
	sparse := models.Company{Name: "Acme"}
	rich := models.Company{
		Name:          "Acme",
		Domain:        strPtr("acme.com"),
		Website:       strPtr("https://acme.com"),
		Description:   strPtr("A widget company"),
		EmployeeCount: strPtr("51-200"),
		LinkedinURL:   strPtr("https://linkedin.com/company/acme"),
	}
	if completenessScore(rich) <= completenessScore(sparse) {
		t.Errorf("expected a fully-enriched company to score higher than a bare one")
	}
}

func TestPickPrimaryCompanyPrefersMostComplete(t *testing.T) {
	sparse := models.Company{Name: "Acme"}
	rich := models.Company{Name: "Acme Inc", Domain: strPtr("acme.com"), Website: strPtr("https://acme.com")}

	primary := pickPrimaryCompany([]models.Company{sparse, rich})
	if primary.Name != "Acme Inc" {
		t.Errorf("expected the richer record to be picked as primary, got %q", primary.Name)
	}
}

func TestMergeStringSlices(t *testing.T) {
	got := mergeStringSlices([]string{"a", "b", ""}, []string{"b", "c"})
	want := map[string]bool{"a": true, "b": true, "c": true}
	if len(got) != len(want) {
		t.Fatalf("mergeStringSlices() = %v, want 3 unique non-empty values", got)
	}
	for _, s := range got {
		if !want[s] {
			t.Errorf("unexpected value %q in merged slice", s)
		}
	}
}
