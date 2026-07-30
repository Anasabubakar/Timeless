package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

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

func TestCompileRoutePattern(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"/api/v1/companies/:id", "/api/v1/companies/abc-123", true},
		{"/api/v1/companies/:id", "/api/v1/companies/abc-123/", true}, // optional trailing slash
		{"/api/v1/companies/:id", "/api/v1/companies/abc/extra", false},
		{"/api/v1/companies/", "/api/v1/companies", true}, // trailing slash on the pattern itself is optional both ways
		{"/api/v1/companies/", "/api/v1/companies/", true},
		{"/api/v1/webhooks/:id/deliveries", "/api/v1/webhooks/xyz/deliveries", true},
		{"/api/v1/webhooks/:id/deliveries", "/api/v1/webhooks/xyz/rotate-secret", false},
		{"/api/v1/companies/dedupe", "/api/v1/companies/dedupe", true},
		{"/api/v1/companies/dedupe", "/api/v1/companies/not-dedupe", false},
	}

	for _, tc := range cases {
		t.Run(tc.pattern+" vs "+tc.path, func(t *testing.T) {
			re := compileRoutePattern(tc.pattern)
			if got := re.MatchString(tc.path); got != tc.want {
				t.Errorf("compileRoutePattern(%q).MatchString(%q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
			}
		})
	}
}

// TestLookupPermissionMatchesEveryCompiledRoute is a completeness check
// over the real table: every entry in routePermissions must be
// resolvable by lookupPermission when fed a plausible concrete path for
// that pattern (":id"-style segments replaced with a sample value).
// Catches a pattern that technically parses but can never actually
// match a real request (e.g. a stray regex metacharacter in a path
// segment that wasn't meant to be one).
func TestLookupPermissionMatchesEveryCompiledRoute(t *testing.T) {
	for key, wantPerm := range routePermissions {
		method, pattern, ok := splitKey(key)
		if !ok {
			t.Fatalf("malformed routePermissions key: %q", key)
		}
		samplePath := strings.ReplaceAll(pattern, ":id", "11111111-1111-1111-1111-111111111111")
		samplePath = strings.ReplaceAll(samplePath, ":provider", "notion")
		samplePath = strings.ReplaceAll(samplePath, ":pageID", "abc123")

		gotPerm, ok := lookupPermission(method, samplePath)
		if !ok {
			t.Errorf("lookupPermission(%q, %q) found no match for routePermissions entry %q", method, samplePath, key)
			continue
		}
		if gotPerm != wantPerm {
			t.Errorf("lookupPermission(%q, %q) = %q, want %q", method, samplePath, gotPerm, wantPerm)
		}
	}
}

func splitKey(key string) (method, path string, ok bool) {
	for i := 0; i < len(key); i++ {
		if key[i] == ' ' {
			return key[:i], key[i+1:], true
		}
	}
	return "", "", false
}

// TestRouteGuardHandleWorksWhenBundledIntoAGroup is the regression test
// for the actual production bug: RouteGuard.Handle runs as one of
// several handlers attached to a single empty-prefix Group (see
// router.Setup's `protected` group) — not as a route's own terminal
// handler. Fiber only reassigns ctx.route to the endpoint's own
// registered pattern once execution reaches that endpoint's Handlers
// slice; while still inside a bundled Group handler, c.Route().Path
// reports the *group's* coarse registration path ("/api/v1"), which is
// never a key in routePermissions — so the old c.Route().Path-based
// implementation denied every single request, regardless of which
// endpoint it hit. This reproduces that exact structure (bundle a stand-in
// for RouteGuard.Handle into an empty-prefix Group ahead of the real
// endpoint, exactly like router.Setup does) and asserts a known,
// permission-mapped route is actually reachable.
func TestRouteGuardHandleWorksWhenBundledIntoAGroup(t *testing.T) {
	rbac := NewRBAC(nil) // logDenial no-ops on a nil db; not exercised on the success path
	guard := NewRouteGuard(rbac)

	app := fiber.New()
	api := app.Group("/api/v1")

	// Mirrors router.Setup: RouteGuard.Handle is bundled into an
	// empty-prefix Group alongside other middleware, not attached
	// directly to the terminal route.
	protected := api.Group("", func(c fiber.Ctx) error {
		// Stand-in for auth/tenant middleware setting Locals before
		// RouteGuard runs — GetOrgID/GetUserID just need non-nil UUIDs
		// so logDenial (if reached) doesn't hit a nil pointer; the
		// permission check itself is bypassed by wildcard below.
		c.Locals("org_id", "11111111-1111-1111-1111-111111111111")
		c.Locals("user_id", "22222222-2222-2222-2222-222222222222")
		return c.Next()
	}, guard.Handle)

	notifications := protected.Group("/notifications")
	notifications.Get("/", func(c fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest("GET", "/api/v1/notifications/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// GET /api/v1/notifications/ is authOnly in routePermissions — it
	// should reach the handler (200), not be denied as "unclassified"
	// (403) the way the old c.Route().Path-based check would have.
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("status = %d, want 200 — RouteGuard incorrectly denied a route bundled into a Group", resp.StatusCode)
	}
}
