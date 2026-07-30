# Security Checklist

Pre-launch checklist. Check items against a real staging deployment,
not just this repo — several of these need a live environment this
sandbox didn't have.

## Authentication
- [x] Passwords hashed with bcrypt
- [x] Brute-force lockout (`MAX_FAILED_LOGINS`/`LOGIN_LOCKOUT_DURATION`)
- [x] Email verification flow exists (`/auth/verify-email`)
- [x] Password reset flow exists, revokes other sessions on success
- [x] MFA/TOTP available, backup codes, disable requires re-auth
- [x] JWT signing key is rotatable (`JWTKeyring`)
- [x] Sessions are durable/revocable (not just a blacklist)
- [ ] Verify `JWT_SECRET` and `CREDENTIALS_ENCRYPTION_KEY` are set to
      real, distinct, strong values in the production environment
      (`Config.Validate()` refuses to boot otherwise — confirm it's
      actually wired to fail the deploy, not just log a warning)

## Authorization
- [x] Default-deny route permission table (RouteGuard)
- [x] Boot-time coverage self-check for unclassified routes
- [x] Org-scoped repository queries throughout
- [x] Owner tier has last-owner removal/demotion protection
- [ ] Confirm `verifyRouteGuardCoverage`'s warning output is actually
      monitored (it logs, it doesn't page anyone)

## API security
- [x] Per-route body size limits
- [x] Per-route rate limiting (Redis-backed, survives multi-instance)
- [x] Request timeout on AI-provider-calling routes
- [x] Consistent DTO-based input validation (`reqbind.JSON`)
- [x] API version header on `/api/v1`

## Secrets
- [x] Credentials encrypted at rest (AES-256-GCM, rotatable)
- [x] No secrets in logs (redacting structured logger)
- [x] No secrets in API responses (`json:"-"` audited)
- [x] `.env.example` free of dead/unused config
- [ ] Confirm the real production `.env` (or secrets manager
      equivalent) has no placeholder values — `Config.Validate()`
      checks common placeholder patterns but can't catch everything

## Database
- [x] All queries parameterized (no raw SQL string concat)
- [x] Mass-assignment closed via DTOs
- [x] Register() transactional (org+user+role provisioning atomic)
- [x] Audit log immutable at the DB level (UPDATE blocked)
- [ ] Confirm automated backups are actually configured on the
      production database (this is a hosting-provider setting, not
      application code)

## File uploads
- [x] MIME type + size + folder allowlisting
- [x] Filename sanitization, path-traversal-safe key generation
- [x] Malware-scan interface exists (`storage.Scanner`)
- [ ] Wire a real scanner (ClamAV daemon or a hosted API) — currently
      `NoopScanner`, documented as a known gap, not silently absent

## Headers / transport
- [x] CSP, HSTS, X-Content-Type-Options, X-Frame-Options,
      Referrer-Policy, Permissions-Policy on every response
- [x] Origin validation on the protected API group and WS upgrade
- [ ] Confirm TLS termination is correctly configured at the
      production load balancer/edge (application-level HSTS assumes
      HTTPS is actually being served)

## Monitoring
- [x] `/health/live` and `/health/ready` both real
- [x] Dead-lettered background jobs alert (log + audit event)
- [x] Security events centrally logged (`LogSecurityEvent`)
- [ ] Wire actual alerting (PagerDuty/Slack/etc.) on top of the log
      lines — logging alone doesn't page anyone

## Dependencies
- [x] Manual review found nothing outdated/vulnerable at current pins
- [ ] `govulncheck ./...` and `npm audit --omit=dev` wired into CI
      (blocked by sandbox network access this session — see
      DEPENDENCY_AUDIT.md)
- [ ] Fiber v3 beta -> stable upgrade evaluated with a live env

## Testing
- [x] Unit tests for RBAC/RouteGuard, JWT keyring, TOTP/MFA,
      security headers, rate limiting, redaction, audit logic
- [x] Regression tests for both critical routing bugs found this pass
- [ ] Full integration test suite against a real Postgres/Redis
      instance (this sandbox's network access to provision one was
      unreliable)
- [ ] An actual penetration test or dynamic scan
