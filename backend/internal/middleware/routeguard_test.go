package middleware

import "testing"

// TestRoutePermissionKnownRoutes spot-checks a representative sample of
// the routePermissions table across every resource area, plus the
// authOnly self-service routes and a route that must not exist. A wrong
// entry here (wrong permission, wrong method, stale pattern after a
// route is renamed) is exactly the class of bug that turns into either
// an accidental privilege escalation or an accidental lockout.
func TestRoutePermissionKnownRoutes(t *testing.T) {
	cases := []struct {
		method, path string
		wantPerm     string
		wantOK       bool
	}{
		{"GET", "/api/v1/companies/", PermCompaniesRead, true},
		{"POST", "/api/v1/companies/", PermCompaniesWrite, true},
		{"DELETE", "/api/v1/companies/:id", PermCompaniesDelete, true},
		{"POST", "/api/v1/companies/dedupe", PermCompaniesWrite, true},
		{"POST", "/api/v1/companies/batch/delete", PermCompaniesDelete, true},

		{"POST", "/api/v1/proposals/generate", PermProposalsGenerate, true},
		{"DELETE", "/api/v1/webhooks/:id", PermWebhooksDelete, true},
		{"GET", "/api/v1/webhooks/:id/deliveries", PermWebhooksRead, true},

		{"POST", "/api/v1/ai/query", PermAIQuery, true},
		{"POST", "/api/v1/onboarding/discovery/run", PermAIQuery, true},

		{"GET", "/api/v1/team/members", PermTeamRead, true},
		{"POST", "/api/v1/team/members", PermTeamManage, true},
		{"DELETE", "/api/v1/team/members/:id", PermTeamManage, true},

		{"POST", "/api/v1/files/upload", PermFilesUpload, true},
		{"DELETE", "/api/v1/files/", PermFilesDelete, true},

		// Self-service: authenticated only, no business permission.
		{"GET", "/api/v1/auth/me", authOnly, true},
		{"PATCH", "/api/v1/profile", authOnly, true},
		{"GET", "/api/v1/notifications/", authOnly, true},
		{"GET", "/api/v1/onboarding/state", authOnly, true},

		// Never registered — must default-deny (ok=false), not silently
		// resolve to some other route's permission.
		{"POST", "/api/v1/companies/:id/archive", "", false},
		{"GET", "/api/v1/admin/impersonate", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			perm, ok := RoutePermission(tc.method + " " + tc.path)
			if ok != tc.wantOK {
				t.Fatalf("RoutePermission(%q) ok = %v, want %v", tc.method+" "+tc.path, ok, tc.wantOK)
			}
			if ok && perm != tc.wantPerm {
				t.Fatalf("RoutePermission(%q) = %q, want %q", tc.method+" "+tc.path, perm, tc.wantPerm)
			}
		})
	}
}

// allDeclaredPermissions mirrors every Perm* constant in permissions.go.
// Kept as an explicit list (not derived from the role slices) because
// several permissions are deliberately Owner/Admin-only — reachable
// only through the "*" wildcard, not listed in any granular role's
// slice — so "does some role's slice contain this string" isn't a valid
// typo check on its own.
var allDeclaredPermissions = []string{
	PermCampaignsRead, PermCampaignsWrite, PermCampaignsDelete,
	PermSponsorsRead, PermSponsorsWrite, PermSponsorsDelete,
	PermCompaniesRead, PermCompaniesWrite, PermCompaniesDelete,
	PermContactsRead, PermContactsWrite, PermContactsDelete,
	PermProposalsRead, PermProposalsWrite, PermProposalsDelete, PermProposalsGenerate,
	PermOutreachRead, PermOutreachWrite, PermOutreachDelete,
	PermAutomationsRead, PermAutomationsWrite, PermAutomationsDelete,
	PermIntegrationsRead, PermIntegrationsWrite, PermIntegrationsDelete,
	PermWebhooksRead, PermWebhooksWrite, PermWebhooksDelete,
	PermAnalyticsRead, PermAIQuery,
	PermSettingsRead, PermSettingsWrite,
	PermUsersRead, PermUsersWrite, PermUsersDelete,
	PermFilesUpload, PermFilesRead, PermFilesDelete,
	PermActivitiesRead, PermActivitiesWrite,
	PermCommunicationsRead, PermCommunicationsWrite, PermCommunicationsDelete,
	PermKnowledgeRead, PermKnowledgeWrite,
	PermNotificationsRead, PermNotificationsWrite,
	PermTeamRead, PermTeamManage,
	PermImportsWrite,
	PermEmailsSend,
	PermAll,
}

// TestRoutePermissionsUseKnownConstants ensures every permission string
// in the table is either the authOnly sentinel or one of the constants
// actually declared in permissions.go — catching a typo'd permission
// string (e.g. "webhook:write" instead of "webhooks:write") that would
// silently make a route unreachable by every role, including Admin,
// since even the wildcard check in satisfiesAll only ever compares
// against the literal "*" and the route's exact required string.
func TestRoutePermissionsUseKnownConstants(t *testing.T) {
	known := make(map[string]bool, len(allDeclaredPermissions))
	for _, p := range allDeclaredPermissions {
		known[p] = true
	}

	for route, perm := range routePermissions {
		if perm == authOnly {
			continue
		}
		if !known[perm] {
			t.Errorf("route %q requires permission %q, which is not a declared permission constant", route, perm)
		}
	}
}
