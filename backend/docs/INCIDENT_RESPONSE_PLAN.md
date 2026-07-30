# Incident Response Plan

## Detection
- `LogSecurityEvent`-backed events (login_failure, permission_denied,
  rate_limit_violation, mfa_disabled, etc.) in the `activities` table —
  query by `entity_type` and `type`.
- `task_dead_lettered` events for background job failures.
- Structured logs (redacted) on stdout — ship to a log aggregator in
  production; not done in this sandbox.

## Triage — is this an incident?
Escalate immediately if you see: a burst of `login_failure` across
many distinct accounts from one IP (credential stuffing), a
`permission_denied` burst for one user (probing), an `mfa_disabled`
or `role_changed` event the account owner doesn't recognize, or any
`unclassified route` RouteGuard denial in production logs (means a
route shipped without being added to the permission table — audit it
immediately, don't just silence the log).

## Containment
1. **Compromised user account**: `POST /auth/sessions/revoke-all` (as
   the user or via an admin action), force a password reset, check
   `activities` for what the session did before revocation.
2. **Compromised integration credential**: `POST
   /integrations/:id/revoke` (wipes Timeless's copy immediately —
   remember this doesn't invalidate the token at the provider; see
   DEPENDENCY_AUDIT.md/SECURITY_ARCHITECTURE.md), then rotate the
   underlying OAuth app secret if the provider supports it.
3. **Compromised JWT_SECRET or CREDENTIALS_ENCRYPTION_KEY**: rotate —
   set a new value, move the old one into `JWT_SECRET_PREVIOUS` /
   `CREDENTIALS_ENCRYPTION_KEY_PREVIOUS` so existing tokens/credentials
   stay valid through the transition, then run the rotate-credentials
   endpoint to re-encrypt everything under the new key and eventually
   drop the retired key from the `_PREVIOUS` list.
4. **Active abuse from an IP/account**: the Redis rate limiter already
   throttles; for a hard block, add IP-level blocking at the edge/load
   balancer (application-level rate limiting isn't a substitute for
   this at scale).

## Eradication & recovery
- Identify root cause via the audit trail (immutable — can't have been
  altered post-hoc, though see the accepted-risk note on DELETE in
  RISK_ASSESSMENT.md).
- Patch the underlying vulnerability, add a regression test (see this
  session's RouteGuard/WS-route fixes for the pattern: reproduce the
  bug in an isolated test before touching the fix, then keep the test
  as a permanent regression guard).
- Restore from backup only if data integrity is actually in question —
  see DISASTER_RECOVERY_PLAN.md.

## Post-incident
- Write up what happened, root cause, timeline, and the fix in a
  postmortem doc (not templated here — genuinely varies per incident).
- Add the specific detection signal that would have caught this
  sooner, if one didn't already exist.

## What's not yet wired
- No automated paging (PagerDuty/Slack/etc.) on top of the security
  event log — someone has to be watching it. Flagged in
  SECURITY_CHECKLIST.md as a pre-launch action item.
