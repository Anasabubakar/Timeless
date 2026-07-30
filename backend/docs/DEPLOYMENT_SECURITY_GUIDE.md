# Deployment Security Guide

## Required environment variables (production)
- `ENVIRONMENT=production` — flips `Config.Validate()` into strict
  mode (rejects weak/placeholder/shared secrets at boot instead of
  warning).
- `JWT_SECRET` — >= 32 random chars, not a placeholder pattern
  (`Config.Validate()` checks common ones).
- `CREDENTIALS_ENCRYPTION_KEY` — set explicitly; do **not** leave it
  unset (it would silently fall back to `JWT_SECRET`, and
  `Config.Validate()` refuses to boot in production without it set
  and distinct from `JWT_SECRET`).
- `DATABASE_URL`, `REDIS_URL` — pointed at production instances with
  TLS where the provider supports it (`sslmode=require` or equivalent
  on the Postgres connection string).
- `FRONTEND_URL`, `ALLOWED_ORIGINS` — the real production origin(s).
  `CORSOrigins()` combines both; get this wrong and either legitimate
  requests get CORS-blocked or the origin allowlist is too permissive.
- `API_PUBLIC_URL` — must match the publicly reachable API URL exactly
  (used to build the OAuth redirect_uri sent to Notion/Apollo; a
  mismatch breaks OAuth entirely, it won't fail open).

## TLS
Application-level `Strict-Transport-Security` header assumes TLS is
actually terminated somewhere in front of it — confirm the load
balancer/edge (not application code) is doing TLS termination and
redirecting HTTP to HTTPS.

## Network
- Postgres and Redis should not be publicly reachable — same VPC/
  private network as the API, or an allowlisted connection.
- MinIO/S3 bucket should not be public-read; file access goes through
  signed, time-limited URLs (`storage.GenerateKey` +
  `PresignedGetObject`), not direct bucket URLs.

## Before first deploy
1. Run through `SECURITY_CHECKLIST.md` — every unchecked item there is
   either a live-environment verification step or a genuine pre-launch
   action item.
2. Confirm automated Postgres backups are enabled at the hosting
   provider (see `DISASTER_RECOVERY_PLAN.md`).
3. Confirm `govulncheck`/`npm audit` run in CI (not done automatically
   by this codebase — wire it into the pipeline).
4. Set real values for every OAuth client ID/secret you intend to use
   (Notion, Apollo) — an unconfigured provider fails gracefully
   (`oauth_not_configured`) rather than crashing, but confirm the ones
   you need are actually set.
5. Decide on and provision bot-detection/WAF at the edge — not
   something this application layer alone provides (see
   `RISK_ASSESSMENT.md` items #3/#4).

## Ongoing
- Rotate `JWT_SECRET`/`CREDENTIALS_ENCRYPTION_KEY` periodically per
  `CREDENTIAL_MANAGEMENT_GUIDE.md` (no fixed cadence is prescribed
  here — align with your org's actual security policy).
- Monitor `/health/ready` from an external uptime check, not just
  `/health` — the latter never reflects a dependency outage.
- Review `SECURITY_RUNBOOK.md`'s "unclassified routes" check
  periodically — a route shipped without RouteGuard classification
  fails closed (safe by default), but should still be fixed promptly
  so the feature actually works.
