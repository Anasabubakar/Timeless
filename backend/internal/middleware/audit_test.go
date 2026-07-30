package middleware

import "testing"

func TestShouldAudit(t *testing.T) {
	cases := []struct {
		name   string
		method string
		status int
		want   bool
	}{
		{"GET is never audited", "GET", 200, false},
		{"OPTIONS is never audited", "OPTIONS", 200, false},
		{"HEAD is never audited", "HEAD", 200, false},
		{"successful POST is audited", "POST", 201, true},
		{"successful DELETE is audited", "DELETE", 204, true},
		{"401 is skipped (never reaches this middleware in practice)", "POST", 401, false},
		{"403 is skipped (RBAC logs its own denial event)", "DELETE", 403, false},
		{"429 is skipped (rate limiter logs its own violation event)", "POST", 429, false},
		{"400 validation failure is skipped (routine, not a security event)", "POST", 400, false},
		{"500 on a mutating request is audited", "DELETE", 500, true},
		{"503 on a mutating request is audited", "PATCH", 503, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldAudit(tc.method, tc.status); got != tc.want {
				t.Errorf("shouldAudit(%q, %d) = %v, want %v", tc.method, tc.status, got, tc.want)
			}
		})
	}
}

func TestMapAction(t *testing.T) {
	cases := []struct {
		method string
		failed bool
		want   string
	}{
		{"POST", false, "created"},
		{"PATCH", false, "updated"},
		{"PUT", false, "updated"},
		{"DELETE", false, "deleted"},
		{"POST", true, "failed_created"},
		{"DELETE", true, "failed_deleted"},
	}

	for _, tc := range cases {
		if got := mapAction(tc.method, tc.failed); got != tc.want {
			t.Errorf("mapAction(%q, %v) = %q, want %q", tc.method, tc.failed, got, tc.want)
		}
	}
}

func TestParseEntity(t *testing.T) {
	cases := []struct {
		path           string
		wantType       string
		wantIDNonEmpty bool
	}{
		{"/api/v1/companies/", "company", false},
		{"/api/v1/companies/11111111-1111-1111-1111-111111111111", "company", true},
		{"/api/v1/campaigns/not-a-uuid", "campaign", false},
		{"/api/v1/activities/", "activity", false},
		{"/api/v1/proposals/", "proposal", false},
	}

	for _, tc := range cases {
		gotType, gotID := parseEntity(tc.path)
		if gotType != tc.wantType {
			t.Errorf("parseEntity(%q) type = %q, want %q", tc.path, gotType, tc.wantType)
		}
		if (gotID != "") != tc.wantIDNonEmpty {
			t.Errorf("parseEntity(%q) id = %q, wantNonEmpty=%v", tc.path, gotID, tc.wantIDNonEmpty)
		}
	}
}

func TestSingularize(t *testing.T) {
	cases := map[string]string{
		"companies":      "company",
		"activities":     "activity",
		"campaigns":      "campaign",
		"proposals":      "proposal",
		"communications": "communication",
	}
	for in, want := range cases {
		if got := singularize(in); got != want {
			t.Errorf("singularize(%q) = %q, want %q", in, got, want)
		}
	}
}
