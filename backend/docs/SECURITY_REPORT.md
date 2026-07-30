# Security Report — Enterprise Hardening Pass

Summary of Part 1 of this engagement: a full-codebase security audit
and hardening pass. See `docs/` in this directory for the supporting
detail (Architecture, Threat Model, Risk Assessment, Checklist,
Incident Response, Credential Management, Disaster Recovery, Runbook,
Deployment Guide, and three focused reviews — dependencies, OWASP API
Top 10, injection/XSS/CSRF).

## Two critical bugs found and fixed

1. **RouteGuard denied every protected request.** `RouteGuard.Handle`
   runs bundled into a Group alongside other middleware, not as a
   route's terminal handler — `c.Route().Path` reports the group's
   coarse registration path ("/api/v1") in that position, not the
   endpoint's actual pattern, so no request ever matched the
   permission table. Confirmed empirically with a standalone Fiber
   reproduction before fixing; fixed by matching the live request path
   against pre-compiled regex patterns instead. Regression-tested.

2. **The WebSocket auth middleware leaked onto unrelated routes.**
   `ws := app.Group("", authMw.HandleWS, ...); ws.Get("/ws", ...)` — an
   empty-prefix Group's middleware attaches at the parent's shared
   prefix, matching every route registered afterward regardless of
   which Group variable registers them. Every notifications/events
   route registered later silently required the WebSocket-specific
   query-param JWT instead of a normal Authorization header. Fixed by
   attaching the middleware directly to `/ws` only. Regression-tested.

## Notable findings, by category

**Authentication**: no email verification, password reset, MFA, brute-
force lockout, or session revocability existed at the start — all
built. JWT signing key is now rotatable. `Register()` was non-
transactional (risk of orphaned orgs/roleless users on partial
failure) and had no org-slug collision handling — both fixed.
Emails weren't normalized before lookup/storage anywhere — fixed.

**Authorization**: `RBACMiddleware` existed but was never attached to
any route, and no organization had any seeded roles — RBAC was
completely inert. Both fixed: default-deny `RouteGuard` covering all
148 routes, five-tier role model with seeded defaults on registration.

**Secrets**: credentials were already well-encrypted at rest; hardened
further with mandatory-in-production key separation, weak/placeholder
secret rejection at boot, and removal of dead config (unused Google
OAuth vars).

**Database**: mass-assignment was possible on essentially every
mutating endpoint (request bodies bound directly into GORM models) —
closed with dedicated DTOs everywhere, including on the audit-log
creation endpoint itself (a client could otherwise forge audit
entries). Added a missing index on the busiest lookup column
(`users.email` — every login).

**AI**: found and fixed a genuine stored/indirect prompt-injection
vector — user-submitted "learned preferences" were concatenated
directly onto the system prompt for all future queries.

**Audit/monitoring**: auth events, team changes, OAuth connections,
permission denials, rate-limit violations, and dead-lettered jobs are
now all centrally logged. Added real readiness/liveness endpoints.

**OAuth**: found and fixed a JWT-verification bug in
`OAuthHandler.Start` that bypassed the key-rotation keyring and had no
algorithm check. Added PKCE as a verified, honest opt-in (confirmed via
current docs that neither Notion nor Apollo supports it — not guessed).
Swept raw provider-error leakage across every handler wrapping an
external call.

**Dependencies**: no critical/high CVEs found at reviewed versions.
Fiber is pinned to a beta with a stable release since available —
flagged, not upgraded blind without a live environment to test
against. Live vulnerability-database scans (`govulncheck`, `npm
audit`) were blocked by sandbox network access this session — must run
in CI going forward.

## Test coverage added
RBAC/RouteGuard (including the bundled-Group regression), JWT keyring,
TOTP/MFA (caught a real backup-code-alphabet bug), security headers,
rate limiting, redaction, mass-assignment DTOs across every hardened
endpoint, AI prompt-injection delimiting, dead-letter handling,
config validation, and the WS-route-leak regression.

## Remaining, honestly

Everything not independently verifiable inside this sandbox is called
out explicitly rather than asserted as done — see `RISK_ASSESSMENT.md`
for the full list with severity. Top items: no live penetration test
has run, no bot-detection/WAF layer is provisioned (deployment-
platform decision), and CI doesn't yet run dependency-vulnerability
scans.

## Recommendation

Application-layer security is substantially hardened and two genuine
critical bugs (one of which meant authorization was silently
non-functional) are fixed and regression-tested. Before a production
launch with paying enterprise customers: run an actual penetration
test, provision edge/WAF protection, wire vulnerability scanning into
CI, and work through the unchecked items in `SECURITY_CHECKLIST.md`.
