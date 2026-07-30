package middleware

import (
	"log"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// authOnly is the sentinel permission value for routes that intentionally
// require nothing beyond authentication — self-service actions on the
// caller's own account/data (profile, sessions, notifications) where a
// business permission like "campaigns:write" doesn't apply.
const authOnly = ""

// routePermissions maps every route registered under the protected API
// group ("METHOD /api/v1/full/pattern", using Fiber's registered pattern
// with :param placeholders — not the interpolated runtime path) to the
// permission required to call it. A route with no entry here is denied
// by default: RouteGuard.Handle fails closed rather than silently
// allowing any authenticated user through, so a route added later
// without updating this table is caught immediately instead of shipping
// as an accidental unauthorized-access hole.
var routePermissions = map[string]string{
	// Auth: self-service, no business permission applies.
	"POST /api/v1/auth/logout":              authOnly,
	"GET /api/v1/auth/me":                   authOnly,
	"POST /api/v1/auth/mfa/enroll":          authOnly,
	"POST /api/v1/auth/mfa/confirm":         authOnly,
	"POST /api/v1/auth/mfa/disable":         authOnly,
	"GET /api/v1/auth/sessions":             authOnly,
	"DELETE /api/v1/auth/sessions/:id":      authOnly,
	"POST /api/v1/auth/sessions/revoke-all": authOnly,

	"GET /api/v1/organizations/current":   PermSettingsRead,
	"PATCH /api/v1/organizations/current": PermSettingsWrite,

	"PATCH /api/v1/profile":         authOnly,
	"POST /api/v1/profile/password": authOnly,

	"GET /api/v1/onboarding/state":     authOnly,
	"PATCH /api/v1/onboarding/state":   authOnly,
	"POST /api/v1/onboarding/complete": authOnly,

	"GET /api/v1/companies/":              PermCompaniesRead,
	"POST /api/v1/companies/":             PermCompaniesWrite,
	"GET /api/v1/companies/:id":           PermCompaniesRead,
	"PATCH /api/v1/companies/:id":         PermCompaniesWrite,
	"DELETE /api/v1/companies/:id":        PermCompaniesDelete,
	"POST /api/v1/companies/dedupe":       PermCompaniesWrite,
	"POST /api/v1/companies/batch/update": PermCompaniesWrite,
	"POST /api/v1/companies/batch/delete": PermCompaniesDelete,

	"GET /api/v1/campaigns/":       PermCampaignsRead,
	"POST /api/v1/campaigns/":      PermCampaignsWrite,
	"GET /api/v1/campaigns/:id":    PermCampaignsRead,
	"PATCH /api/v1/campaigns/:id":  PermCampaignsWrite,
	"DELETE /api/v1/campaigns/:id": PermCampaignsDelete,

	"GET /api/v1/sponsors/":              PermSponsorsRead,
	"POST /api/v1/sponsors/":             PermSponsorsWrite,
	"GET /api/v1/sponsors/:id":           PermSponsorsRead,
	"PATCH /api/v1/sponsors/:id":         PermSponsorsWrite,
	"DELETE /api/v1/sponsors/:id":        PermSponsorsDelete,
	"PATCH /api/v1/sponsors/:id/stage":   PermSponsorsWrite,
	"POST /api/v1/sponsors/batch/update": PermSponsorsWrite,
	"POST /api/v1/sponsors/batch/delete": PermSponsorsDelete,

	"GET /api/v1/contacts/":              PermContactsRead,
	"POST /api/v1/contacts/":             PermContactsWrite,
	"GET /api/v1/contacts/:id":           PermContactsRead,
	"PATCH /api/v1/contacts/:id":         PermContactsWrite,
	"DELETE /api/v1/contacts/:id":        PermContactsDelete,
	"POST /api/v1/contacts/batch/update": PermContactsWrite,
	"POST /api/v1/contacts/batch/delete": PermContactsDelete,

	"GET /api/v1/activities/":  PermActivitiesRead,
	"POST /api/v1/activities/": PermActivitiesWrite,

	"GET /api/v1/sequences/":            PermOutreachRead,
	"POST /api/v1/sequences/":           PermOutreachWrite,
	"GET /api/v1/sequences/:id":         PermOutreachRead,
	"PATCH /api/v1/sequences/:id":       PermOutreachWrite,
	"DELETE /api/v1/sequences/:id":      PermOutreachDelete,
	"POST /api/v1/sequences/:id/enroll": PermOutreachWrite,

	"GET /api/v1/proposals/":          PermProposalsRead,
	"POST /api/v1/proposals/":         PermProposalsWrite,
	"POST /api/v1/proposals/generate": PermProposalsGenerate,
	"GET /api/v1/proposals/:id":       PermProposalsRead,
	"PATCH /api/v1/proposals/:id":     PermProposalsWrite,
	"DELETE /api/v1/proposals/:id":    PermProposalsDelete,

	"GET /api/v1/integrations/":                       PermIntegrationsRead,
	"GET /api/v1/integrations/dashboard":              PermIntegrationsRead,
	"GET /api/v1/integrations/zapier/apps":            PermIntegrationsRead,
	"POST /api/v1/integrations/":                      PermIntegrationsWrite,
	"GET /api/v1/integrations/:id":                    PermIntegrationsRead,
	"PATCH /api/v1/integrations/:id":                  PermIntegrationsWrite,
	"DELETE /api/v1/integrations/:id":                 PermIntegrationsDelete,
	"POST /api/v1/integrations/:id/revoke":            PermIntegrationsWrite,
	"POST /api/v1/integrations/:id/sync":              PermIntegrationsWrite,
	"POST /api/v1/integrations/:provider/connect":     PermIntegrationsWrite,
	"PATCH /api/v1/integrations/notion/pages/:pageID": PermIntegrationsWrite,
	"POST /api/v1/integrations/rotate-credentials":    PermIntegrationsWrite,

	"GET /api/v1/automations/":             PermAutomationsRead,
	"POST /api/v1/automations/":            PermAutomationsWrite,
	"GET /api/v1/automations/:id":          PermAutomationsRead,
	"PATCH /api/v1/automations/:id":        PermAutomationsWrite,
	"DELETE /api/v1/automations/:id":       PermAutomationsDelete,
	"PATCH /api/v1/automations/:id/toggle": PermAutomationsWrite,

	"GET /api/v1/webhooks/":                   PermWebhooksRead,
	"POST /api/v1/webhooks/":                  PermWebhooksWrite,
	"GET /api/v1/webhooks/:id":                PermWebhooksRead,
	"PATCH /api/v1/webhooks/:id":              PermWebhooksWrite,
	"DELETE /api/v1/webhooks/:id":             PermWebhooksDelete,
	"POST /api/v1/webhooks/:id/rotate-secret": PermWebhooksWrite,
	"POST /api/v1/webhooks/:id/test":          PermWebhooksWrite,
	"GET /api/v1/webhooks/:id/deliveries":     PermWebhooksRead,

	"GET /api/v1/communications/":       PermCommunicationsRead,
	"POST /api/v1/communications/":      PermCommunicationsWrite,
	"GET /api/v1/communications/stats":  PermCommunicationsRead,
	"GET /api/v1/communications/:id":    PermCommunicationsRead,
	"PATCH /api/v1/communications/:id":  PermCommunicationsWrite,
	"DELETE /api/v1/communications/:id": PermCommunicationsDelete,

	"GET /api/v1/analytics/dashboard":        PermAnalyticsRead,
	"GET /api/v1/analytics/pipeline":         PermAnalyticsRead,
	"GET /api/v1/analytics/activity":         PermAnalyticsRead,
	"GET /api/v1/analytics/timeseries":       PermAnalyticsRead,
	"GET /api/v1/analytics/funnel":           PermAnalyticsRead,
	"GET /api/v1/analytics/velocity":         PermAnalyticsRead,
	"GET /api/v1/analytics/export/sponsors":  PermAnalyticsRead,
	"GET /api/v1/analytics/export/campaigns": PermAnalyticsRead,

	"POST /api/v1/ai/query":       PermAIQuery,
	"GET /api/v1/ai/agents":       PermAIQuery,
	"POST /api/v1/ai/outcomes":    PermAIQuery,
	"GET /api/v1/ai/outcomes":     PermAIQuery,
	"POST /api/v1/ai/feedback":    PermAIQuery,
	"POST /api/v1/ai/preferences": PermAIQuery,
	"GET /api/v1/ai/preferences":  PermAIQuery,

	"POST /api/v1/onboarding/discovery/run":    PermAIQuery,
	"POST /api/v1/onboarding/discovery/select": PermAIQuery,
	"POST /api/v1/onboarding/goals/recommend":  PermAIQuery,
	"POST /api/v1/onboarding/goals/plan":       PermAIQuery,
	"POST /api/v1/onboarding/goals/approve":    PermAIQuery,

	"GET /api/v1/knowledge/search":              PermKnowledgeRead,
	"GET /api/v1/knowledge/memories":            PermKnowledgeRead,
	"POST /api/v1/knowledge/memories":           PermKnowledgeWrite,
	"POST /api/v1/knowledge/nodes":              PermKnowledgeWrite,
	"POST /api/v1/knowledge/edges":              PermKnowledgeWrite,
	"GET /api/v1/knowledge/nodes/:id/neighbors": PermKnowledgeRead,

	"POST /api/v1/files/upload": PermFilesUpload,
	"DELETE /api/v1/files/":     PermFilesDelete,

	"GET /api/v1/search": authOnly, // spans multiple resource types; results are still org-scoped

	"POST /api/v1/import/companies": PermImportsWrite,
	"POST /api/v1/import/contacts":  PermImportsWrite,
	"POST /api/v1/import/sponsors":  PermImportsWrite,

	"GET /api/v1/team/members":             PermTeamRead,
	"POST /api/v1/team/members":            PermTeamManage,
	"PATCH /api/v1/team/members/:id/roles": PermTeamManage,
	"DELETE /api/v1/team/members/:id":      PermTeamManage,
	"GET /api/v1/team/roles":               PermTeamRead,

	"POST /api/v1/emails/send":          PermEmailsSend,
	"POST /api/v1/emails/send-direct":   PermEmailsSend,
	"POST /api/v1/emails/send-template": PermEmailsSend,

	"GET /api/v1/events": authOnly,

	"GET /api/v1/notifications/":            authOnly,
	"GET /api/v1/notifications/count":       authOnly,
	"POST /api/v1/notifications/read-all":   authOnly,
	"PATCH /api/v1/notifications/:id/read":  authOnly,
	"DELETE /api/v1/notifications/:id":      authOnly,
	"GET /api/v1/notifications/preferences": authOnly,
	"PUT /api/v1/notifications/preferences": authOnly,
}

// RoutePermission exposes routePermissions for the startup coverage
// self-check in router.Setup — it needs to know what RouteGuard would
// require for a given "METHOD /path" key without duplicating the table.
func RoutePermission(methodAndPath string) (string, bool) {
	perm, ok := routePermissions[methodAndPath]
	return perm, ok
}

// compiledRoute is a routePermissions entry with its path pattern
// pre-compiled to a regexp, so matching a live request path doesn't
// require fiber's own Ctx.Route() — see the comment on Handle for why
// that's unusable here.
type compiledRoute struct {
	method string
	path   string
	perm   string
	re     *regexp.Regexp
}

// compiledRoutes is built once at package init from routePermissions —
// the single source of truth stays that map; this is purely a derived
// index for fast, correct lookup at request time.
var compiledRoutes = compileRoutePermissions(routePermissions)

func compileRoutePermissions(perms map[string]string) []compiledRoute {
	out := make([]compiledRoute, 0, len(perms))
	for key, perm := range perms {
		method, path, ok := strings.Cut(key, " ")
		if !ok {
			panic("routeguard: malformed routePermissions key (want \"METHOD /path\"): " + key)
		}
		out = append(out, compiledRoute{method: method, path: path, perm: perm, re: compileRoutePattern(path)})
	}
	return out
}

// compileRoutePattern turns a fiber route pattern ("/api/v1/companies/:id")
// into an anchored regexp that matches the same interpolated runtime paths
// fiber itself would ("/api/v1/companies/abc-123") — ":name" segments
// become a single-segment wildcard, everything else is matched literally.
// The trailing slash is made optional: fiber's default (non-strict)
// routing treats "/api/v1/companies" and "/api/v1/companies/" as the same
// route, and callers (including this app's own frontend) don't always
// include it — the permission lookup has to be exactly as tolerant or a
// perfectly legitimate request with/without a trailing slash falsely
// looks "unclassified" and gets denied.
func compileRoutePattern(pattern string) *regexp.Regexp {
	trimmed := strings.TrimSuffix(pattern, "/")
	segments := strings.Split(trimmed, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			segments[i] = `[^/]+`
		} else {
			segments[i] = regexp.QuoteMeta(seg)
		}
	}
	return regexp.MustCompile("^" + strings.Join(segments, "/") + "/?$")
}

// lookupPermission finds the routePermissions entry matching an incoming
// request, if any.
func lookupPermission(method, path string) (perm string, ok bool) {
	for _, r := range compiledRoutes {
		if r.method == method && r.re.MatchString(path) {
			return r.perm, true
		}
	}
	return "", false
}

// RouteGuard enforces the routePermissions table as the last link in the
// protected middleware chain.
type RouteGuard struct {
	rbac *RBACMiddleware
}

func NewRouteGuard(rbac *RBACMiddleware) *RouteGuard {
	return &RouteGuard{rbac: rbac}
}

// Handle looks up the matched route's required permission and enforces
// it. Routes missing from routePermissions are denied — default-deny,
// not default-allow — since an omission here means nobody decided what
// this route should require, and "nobody decided" must never mean
// "anyone can call it."
//
// This intentionally does NOT use c.Route().Path: RouteGuard.Handle runs
// as one of several handlers bundled into router.Setup's `protected`
// group (a single fiber Use-registration covering every route under
// /api/v1). Fiber only reassigns ctx.route to the terminal, endpoint-
// specific route once execution actually reaches that route's own
// Handlers slice (see fiber's Ctx.Next()/App.next()) — while still
// inside a Use-bundled handler like this one, c.Route() reports the
// bundle's own coarse registration path ("/api/v1"), not the endpoint
// being called. That silently 403'd every single protected request
// ("this route has no authorization policy configured") regardless of
// which endpoint it hit, since "GET /api/v1" (etc.) was never a key in
// routePermissions. Matching the live request path against the
// pre-compiled patterns below sidesteps that entirely.
func (g *RouteGuard) Handle(c fiber.Ctx) error {
	method, path := c.Method(), c.Path()
	perm, ok := lookupPermission(method, path)
	if !ok {
		key := method + " " + path
		log.Printf("routeguard: DENY %s — no authorization policy registered for this route", key)
		g.rbac.logDenial(c, GetUserID(c), GetOrgID(c), "unclassified route: "+key)
		return fiber.NewError(fiber.StatusForbidden, "this route has no authorization policy configured")
	}

	if perm == authOnly {
		return c.Next()
	}

	return g.rbac.Require(perm)(c)
}
