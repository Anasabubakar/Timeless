# Security Runbook

Operational how-tos. For "what happened and what do I do about it
right now," see `INCIDENT_RESPONSE_PLAN.md` — this doc is the
reference for routine and semi-routine tasks.

## Rotate JWT_SECRET
1. Generate a new strong secret (`openssl rand -hex 32`).
2. Set `JWT_SECRET_PREVIOUS` to include the *current* value (prepend
   if others already exist).
3. Set `JWT_SECRET` to the new value.
4. Deploy. `JWTKeyring` resolves both old and new tokens during the
   transition (see CREDENTIAL_MANAGEMENT_GUIDE.md for the retention
   window before removing the old value entirely).

## Rotate CREDENTIALS_ENCRYPTION_KEY
Same pattern as above, then call:
```
POST /api/v1/integrations/rotate-credentials
```
(requires `integrations:write`) to re-encrypt every stored credential
under the new key. Check the response's `rotated`/`checked` counts to
confirm coverage.

## Investigate a suspicious login
```sql
SELECT * FROM activities
WHERE entity_type = 'auth' AND type IN ('login_failure', 'login_success', 'login_blocked')
  AND user_id = '<uuid>'
ORDER BY created_at DESC LIMIT 50;
```
Cross-reference `ip_address` and `metadata->>'reason'`.

## Force-logout a user
```
POST /api/v1/auth/sessions/revoke-all
```
(as that user, or build an admin-scoped variant if needed — not
currently exposed as an admin-on-behalf-of action; today it's
self-service only).

## Check for unclassified routes
Boot logs contain a line per route RouteGuard doesn't recognize:
```
router: WARNING — <METHOD> <path> has no RouteGuard permission entry...
```
Any occurrence in production logs is a same-day fix: add the route to
`routePermissions` in `internal/middleware/routeguard.go` with the
correct permission (or `authOnly` if it's genuinely self-service), or
to `publicAPIRoutes` in `router.go` if it's meant to be public.

## Check for dead-lettered background jobs
```sql
SELECT * FROM activities
WHERE entity_type = 'background_job' AND type = 'task_dead_lettered'
ORDER BY created_at DESC LIMIT 50;
```
`metadata->>'task_type'` and `metadata->>'error'` identify what failed
and why.

## Verify readiness/liveness
```
curl https://<host>/health/live   # process up?
curl https://<host>/health/ready  # DB + Redis reachable? (503 if not)
```

## Audit a specific org's team changes
```sql
SELECT * FROM activities
WHERE entity_type = 'team' AND organization_id = '<uuid>'
ORDER BY created_at DESC;
```

## Run the dependency/security checks that need CI
```
cd backend && govulncheck ./...
cd frontend && npm audit --omit=dev
```
Neither ran reliably in this development sandbox (network access to
the relevant registries was intermittent) — wire both into CI per
DEPENDENCY_AUDIT.md's follow-up section.
