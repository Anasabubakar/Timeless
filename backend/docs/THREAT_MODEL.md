# Threat Model

STRIDE-style pass over Timeless's actual architecture (multi-tenant
SaaS: Go/Fiber backend, Next.js frontend, Postgres, Redis, MinIO,
asynq workers, third-party OAuth integrations). Not exhaustive — a
living document, updated as the system changes.

## Assets

- Customer org data (CRM records, sponsor pipelines, meetings, notes)
- User credentials (passwords, MFA secrets, session tokens)
- Integration credentials (Notion/Apollo OAuth tokens, Zapier tokens,
  webhook secrets)
- Audit trail (who did what — itself a target: an attacker who
  compromises an account wants to also erase the evidence)
- AI provider API keys and prompt/response content

## Actors

- Anonymous internet user (no account)
- Authenticated user, any role tier, any org (the "malicious insider
  from a different tenant" case — the primary multi-tenant threat)
- Authenticated user, low-privilege role, same org (privilege
  escalation within a tenant)
- Compromised third-party integration (Notion/Apollo/Zapier acting
  maliciously or being itself compromised)
- Operator/admin with production access (insider threat, or a
  compromised operator credential)

## STRIDE

### Spoofing
- **Identity**: mitigated by JWT signature verification
  (algorithm-pinned to HMAC, keyring-resolved) + bcrypt password
  checks + MFA. The `OAuthHandler.Start` JWT-verification bug (fixed
  this pass) was specifically a spoofing-adjacent risk — a token
  signed under a rotated-out key wasn't being resolved correctly.
- **Integration identity**: OAuth state parameter prevents a third
  party from injecting their own authorization code into a victim's
  flow (classic OAuth CSRF).

### Tampering
- **Request tampering**: DTOs with `reqbind.JSON`'s
  `DisallowUnknownFields` close the mass-assignment path (a client
  can't set `organization_id`/`id`/bookkeeping fields it shouldn't
  control).
- **Audit log tampering**: DB trigger blocks UPDATE on `activities` —
  an attacker (or a bug) can't quietly edit the trail after the fact.
  DELETE remains possible (retention purge needs it) — a residual risk
  if an attacker gets direct DB access; mitigated by the DB credential
  itself being a high-value secret protected like any other.
- **Route tampering**: RouteGuard's default-deny table means a route
  added later without explicit classification fails closed instead of
  silently becoming reachable.

### Repudiation
- Every security-relevant event (login, logout, MFA, password change,
  role change, permission denial, rate-limit violation, OAuth connect,
  dead-lettered job) is logged via `LogSecurityEvent` with actor, IP,
  and metadata — an actor can't plausibly claim "I didn't do that" for
  anything covered.
- Gap: read-only actions (GET requests) are not audited at all — a
  data-exfiltration-via-read scenario (an attacker with a valid but
  narrowly-scoped session pulling large amounts of data via legitimate
  read endpoints) wouldn't show up in the trail beyond rate-limit
  counters. Documented as a known gap, not silently absent.

### Information Disclosure
- Cross-tenant: every repository query is org-scoped; confirmed no
  handler trusts a client-supplied org_id over the JWT's.
- Secrets: encrypted at rest (credentials), redacted in logs
  (structured logger), never serialized in API responses (json:"-" on
  every sensitive model field), never leaked via raw provider error
  text (swept across every handler wrapping an external call).
- Field-level: team member roles hidden from anyone without
  `team:manage` — the roster is visible, who-has-elevated-access isn't.

### Denial of Service
- Per-route rate limiting (Redis sliding window + burst), request
  body size caps, an 8000-char cap on AI queries specifically (token
  cost control), request timeouts on AI-provider-calling routes.
- No WAF/edge DDoS layer — that's infrastructure this sandbox can't
  provision; flagged for the deployment platform (Cloudflare/Vercel/
  similar) to provide, not something application code alone can fully
  own.

### Elevation of Privilege
- RBAC is default-deny at the route level (RouteGuard) and
  permission-based, not role-name-based, so a new route can't
  accidentally inherit broader access than intended.
- Owner tier has explicit last-owner protection (can't remove/demote
  the only Owner in an org) — closes a "strand the org with no one who
  can grant Owner back" self-lockout, and incidentally also closes a
  "demote the only Owner then re-invite yourself as Owner" attack
  shape for a compromised Admin account.
- MFA disable requires re-entering the current password — a hijacked,
  already-authenticated session alone isn't enough to strip account
  protection.

## Top residual risks (see SECURITY_REPORT.md for full list + severity)

1. No live penetration test has been run against a deployed instance —
   everything above is code-level analysis and unit/integration
   testing within this sandbox, not a dynamic attack simulation.
2. No bot-detection/CAPTCHA layer (OWASP API6).
3. Fiber pinned to a beta release with a stable version since shipped
   — flagged, not upgraded blind without a live environment to
   regression-test against.
4. Read-heavy data exfiltration via legitimate, authenticated access
   isn't specifically detected (only rate-limited).
