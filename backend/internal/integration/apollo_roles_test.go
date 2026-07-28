package integration

import "testing"

func TestTargetRolesHasNoDuplicates(t *testing.T) {
	seen := make(map[string]bool, len(TargetRoles))
	for _, role := range TargetRoles {
		if seen[role] {
			t.Errorf("duplicate role in TargetRoles: %q", role)
		}
		seen[role] = true
	}
}

func TestTargetRolesIncludesSpecRequiredRoles(t *testing.T) {
	required := []string{
		"Founder", "CEO", "Co-Founder", "CMO", "Marketing Director",
		"Head of Partnerships", "Partnership Manager", "Developer Relations",
		"Community Lead", "Brand Lead", "Country Manager", "Regional Manager",
		"Events Manager", "Communications Lead",
	}
	present := make(map[string]bool, len(TargetRoles))
	for _, r := range TargetRoles {
		present[r] = true
	}
	for _, want := range required {
		if !present[want] {
			t.Errorf("TargetRoles is missing required role %q", want)
		}
	}
}

func TestMaxEmailRevealsPerCompanyIsPositive(t *testing.T) {
	if maxEmailRevealsPerCompany <= 0 {
		t.Errorf("maxEmailRevealsPerCompany = %d, must be positive to reveal any emails at all", maxEmailRevealsPerCompany)
	}
}
