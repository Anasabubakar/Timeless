# OWASP API Security Top 10 (2023) — Checklist

Reviewed against the router (`internal/router/router.go`), RouteGuard's
permission table, and the handlers/services touched across this
project's security-hardening pass. "Status" reflects the codebase as
of this commit, not a live penetration test — see
`backend/docs/DEPENDENCY_AUDIT.md` for what still needs a real
environment to verify.

## API1:2023 — Broken Object Level Authorization
**Status: addressed.** Every repository method that fetches/updates/
deletes a specific record scopes the query by `organization_id`
(confirmed across Company/Contact/Sponsor/Campaign/Proposal/
Integration/Webhook/Automation repositories), so an authenticated user
from org A can't reach org B's records by guessing an ID — the WHERE
clause itself excludes them, not just a post-fetch check. `RevokeSession`
additionally verifies the target session belongs to the requesting user
before revoking it.

## API2:2023 — Broken Authentication
**Status: addressed.** JWT access/refresh tokens with a rotating,
kid-tagged signing keyring; bcrypt password hashing; brute-force
lockout after `MAX_FAILED_LOGINS`; TOTP MFA with hashed backup codes;
durable, revocable sessions (not just a Redis blacklist); password
reset/change both revoke every other session. See
`docs/SECURITY_ARCHITECTURE.md` (once written) for the full auth
design.

## API3:2023 — Broken Object Property Level Authorization
**Status: addressed.** Every mutating handler for a core entity
(Company/Contact/Sponsor/Campaign/Automation/Activity/...) binds
through a dedicated `*Input` DTO listing exactly the client-writable
fields — not the GORM model directly — closing the mass-assignment gap
where a request body could set `id`, `organization_id`, `created_at`,
or a bookkeeping field like `run_count`. `TeamHandler.ListMembers`
additionally redacts each member's role list from anyone without
`team:manage`, since role visibility itself is sensitive (reconnaissance
value for social engineering).

## API4:2023 — Unrestricted Resource Consumption
**Status: addressed.** Per-route request body size caps
(`MaxBodySize`), a Redis-backed sliding-window rate limiter with
per-route policies (auth endpoints, AI endpoints, general API, OAuth,
webhooks all have distinct limits — see `internal/middleware/
ratelimit.go`), a request timeout on AI-provider-calling routes
(`aiRequestTimeout`), and an 8000-character cap on the AI query field
specifically to bound token cost per request.

## API5:2023 — Broken Function Level Authorization
**Status: addressed** (with one caveat below). `RouteGuard` is a
default-deny mapping of every registered route to the exact permission
it requires — a route with no table entry is denied outright rather
than silently allowed, and a boot-time self-check
(`verifyRouteGuardCoverage`) warns if a route exists that RouteGuard
doesn't know about. **Caveat**: this was recently fixed after
discovering `RouteGuard.Handle`'s original `c.Route().Path`-based
lookup silently failed when bundled into a Group (see the routeguard.go
fix commit) — regression-tested, but the class of bug (a Fiber routing
assumption not holding under a different registration structure) is
worth extra scrutiny on any future middleware-chain change.

## API6:2023 — Unrestricted Access to Sensitive Business Flows
**Status: partially addressed.** Rate limiting covers the obvious
abuse vectors (registration, login, password reset, AI queries). No
dedicated bot-detection/CAPTCHA layer exists — flagged as an
infra-level item needing a real deployment to evaluate (see
DEPENDENCY_AUDIT.md's "Follow-up" section for the equivalent caveat on
other infra-only items).

## API7:2023 — Server Side Request Forgery
**Status: addressed for what exists.** The only outbound requests this
app makes to attacker-influenceable URLs are OAuth token exchanges —
and those always target a fixed, provider-configured `TokenURL` from
`OAuthProvider`, never a client-supplied URL. No handler accepts an
arbitrary URL from a request body and fetches it server-side.

## API8:2023 — Security Misconfiguration
**Status: addressed.** Security headers middleware (CSP, HSTS,
X-Content-Type-Options, X-Frame-Options, Referrer-Policy,
Permissions-Policy), origin-validated CORS, `Config.Validate()`
refusing to boot in production with a placeholder/weak/shared
JWT/credentials secret, and `.env.example` kept free of dead
configuration (the unused Google OAuth vars were removed rather than
left as misleading dead config).

## API9:2023 — Improper Inventory Management
**Status: addressed.** `routePermissions` plus `publicAPIRoutes` in
router.go, cross-checked by script against every actual route
registration during the RouteGuard rollout (148 routes, zero
unaccounted for) and re-verified at every boot via
`verifyRouteGuardCoverage`. API versioning header
(`middleware.WithAPIVersion`) is present on the `/api/v1` group.

## API10:2023 — Unsafe Consumption of APIs
**Status: addressed for the identified cases.** Every handler wrapping
a live external API call (Notion, Apollo, Zapier MCP, SMTP/SendGrid)
was audited for raw error passthrough — provider error text is now
logged server-side and replaced with a generic client-facing message,
since a provider's raw response could carry more detail than should
reach an API consumer. `learned_context` fed back into AI system
prompts (sourced from prior AI interactions, which are themselves
downstream of AI-provider output) is explicitly delimited as untrusted
content rather than concatenated as trusted instructions.

## Not yet independently verified in this sandbox
- Live penetration testing / fuzzing against a running instance
- A real CAPTCHA/bot-detection layer for API6
- The dependency-vulnerability-database checks noted in
  DEPENDENCY_AUDIT.md (govulncheck/npm audit were unreachable here)
