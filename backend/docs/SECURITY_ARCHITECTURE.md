# Security Architecture

How Timeless's security controls fit together. For the audit trail of
individual findings/fixes, see `SECURITY_REPORT.md`. For per-topic
detail, see the other docs in this directory.

## Layers, outside in

```
Browser / API client
   |  Authorization: Bearer <jwt>   (no cookies anywhere — see INJECTION_XSS_CSRF_REVIEW.md)
   v
Security headers (CSP, HSTS, X-Frame-Options, ...) — every response
   |
Origin validation (ValidateOrigin / ValidateOriginAlways) — protected group + WebSocket upgrade
   |
Rate limiting (Redis sliding window, per-route policy) — internal/middleware/ratelimit.go
   |
Auth middleware (JWT verification via JWTKeyring, kid-aware) — internal/middleware/auth.go
   |
Tenant middleware (resolves + enforces org context) — internal/middleware/tenant.go
   |
Audit logging (2xx mutations + 5xx failures) — internal/middleware/audit.go
   |
RouteGuard (default-deny permission enforcement) — internal/middleware/routeguard.go
   |
Handler (DTO binding + validation via reqbind, never the raw model)
   |
Service (business logic, security events via LogSecurityEvent)
   |
Repository (org-scoped, parameterized GORM queries)
   |
Postgres (encrypted-at-rest credentials, immutable activities table)
```

## Identity & sessions

- **Passwords**: bcrypt, `DefaultCost`. Reset/change both revoke every
  other session and email a change notification.
- **Tokens**: short-lived (15m) access JWTs, longer-lived refresh JWTs
  persisted as real session rows (`RefreshToken` model — hash only,
  never the token itself), not just a Redis blacklist entry. Refresh
  rotates the token and revokes the old session.
- **MFA**: TOTP (RFC 6238, stdlib-only, no third-party crypto
  dependency), encrypted secret at rest, bcrypt-hashed backup codes,
  a signed short-lived ticket (`mfa_pending`) binds the MFA step to an
  already-verified password so completing MFA can't be decoupled from
  proving the password.
- **Lockout**: `MAX_FAILED_LOGINS` consecutive failures locks the
  account for `LOGIN_LOCKOUT_DURATION`; the counter resets on success.
- **Key rotation**: `JWTKeyring`/`CredentialCipher` both support a
  current key plus a list of retired keys — rotating the active secret
  doesn't invalidate tokens/credentials signed/encrypted under the
  previous one.

## Authorization

Every route under `/api/v1` is either in `publicAPIRoutes` (public,
verified another way — OAuth state, HMAC webhook signature) or has an
exact entry in `RouteGuard`'s permission table — a route with neither
is denied by default. Five role tiers (Owner/Admin/Manager/Member/
Guest) map to permission sets in `internal/middleware/permissions.go`;
Owner has extra protection (can't be the last one removed/demoted from
an org).

## Data protection

- OAuth tokens and integration credentials: AES-256-GCM
  (`security.CredentialCipher`), key-rotatable.
- Audit trail: `activities` table, DB-trigger-enforced immutable
  (UPDATE blocked), retention purge opt-in via
  `AUDIT_LOG_RETENTION_DAYS` (default: keep forever).
- Every DTO explicitly excludes id/organization_id/created_at/
  bookkeeping fields — a request body can only ever set what a
  handler's `*Input` struct declares.

## Integrations

OAuth (Notion, Apollo): state-based CSRF protection (Redis, one-time,
10-minute TTL), PKCE available as an opt-in per provider (off for both
current providers — neither documents support, verified via their
current docs), redirect URIs are always server-configured constants
(never derived from client input). See `docs/DEPENDENCY_AUDIT.md` and
the OAuth-hardening commits for the full history, including a critical
JWT-verification bug found and fixed in `OAuthHandler.Start`.

## Observability

Structured, redacting logger (`internal/logging`) strips tokens/
passwords/secrets before anything reaches stdout. Security events
(login/logout/MFA/password changes, permission denials, rate-limit
violations, team role changes, OAuth connections, dead-lettered
background jobs) all flow through one shared primitive,
`middleware.LogSecurityEvent`. `/health/live` and `/health/ready`
distinguish "process is up" from "dependencies are reachable".
