package integration

import "testing"

func TestFindBestRoleMatch(t *testing.T) {
	people := []apolloPerson{
		{ID: "1", Title: "Chief Executive Officer"},
		{ID: "2", Title: "VP of Partnerships"},
		{ID: "3", Title: "Head of Marketing"},
	}

	assigned := map[string]bool{}
	match := findBestRoleMatch("Head of Partnerships", people, assigned)
	if match == nil {
		t.Fatalf("expected a keyword-overlap match for %q", "Head of Partnerships")
	}
	if match.ID != "2" {
		t.Errorf("expected person 2 (VP of Partnerships) to match, got %+v", match)
	}
}

func TestFindBestRoleMatchSkipsAssigned(t *testing.T) {
	people := []apolloPerson{{ID: "1", Title: "Founder"}}
	assigned := map[string]bool{"1": true}

	if match := findBestRoleMatch("Founder", people, assigned); match != nil {
		t.Errorf("expected no match once the only candidate is already assigned, got %+v", match)
	}
}

func TestFindBestRoleMatchNoneFound(t *testing.T) {
	people := []apolloPerson{{ID: "1", Title: "Software Engineer"}}
	if match := findBestRoleMatch("Head of Partnerships", people, map[string]bool{}); match != nil {
		t.Errorf("expected no match for an unrelated title, got %+v", match)
	}
}

func TestConfidenceFromEmailStatus(t *testing.T) {
	cases := map[string]float64{
		"verified":    0.95,
		"guessed":     0.4,
		"unverified":  0.3,
		"unavailable": 0.5,
		"":            0.5,
	}
	for status, want := range cases {
		if got := confidenceFromEmailStatus(status); got != want {
			t.Errorf("confidenceFromEmailStatus(%q) = %v, want %v", status, got, want)
		}
	}
}
