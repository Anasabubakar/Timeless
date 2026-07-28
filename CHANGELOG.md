# Changelog

## Integrations: Zapier, Notion, Apollo (production-grade rewrite)

This entry covers the session that took the integration layer from a
minimal connectivity check into a production-grade system: real OAuth
with refresh, real webhooks, incremental sync, automatic retry/recovery,
duplicate merging, credential rotation, and a live observability
dashboard. See [backend/docs/INTEGRATIONS.md](backend/docs/INTEGRATIONS.md)
for the full architecture writeup.

### Added — Zapier

- Agentic-mode discovery (`list_enabled_zapier_actions`) with automatic
  fallback to classic-mode `tools/list` grouping.
- `ExecuteAction` — the entry point other services use to try Zapier
  before a native client.
- A safe read-only sync pass that only ever auto-invokes zero-argument,
  read-verb-matching tools, never anything that looks like it could
  write/send/delete.

### Added — Notion

- Full OAuth 2.0 flow with refresh-token support (previously assumed not
  to exist — Notion's current docs confirm it's real).
- Schema/database discovery against the 2025-09-03 API (databases split
  into data sources) with no hardcoded database/property names.
- Incremental sync via a stored watermark, so steady-state re-syncs only
  read what changed.
- A real-time webhook receiver with signature verification.
- Conflict-safe write-back that refuses to overwrite a page edited in
  Notion since it was last read.

### Added — Apollo

- Organization enrichment by domain (industry, employee count, revenue,
  funding, technologies, location).
- Role-based decision-maker discovery against a fixed checklist (Founder,
  CEO, Co-Founder, CMO, Head of Partnerships, and more), reporting
  `Available: false` rather than fabricating data when nobody matches.
- Corrected the people-search endpoint to `mixed_people/api_search` per
  current docs.

### Added — Reliability & observability

- `sync_runs` history table plus a live dashboard (connection health,
  recent runs, 24h failure counts, pending jobs).
- Distinct handling for expired auth (refresh-then-reconnect), rate
  limits (provider-aware backoff), and deleted integrations (stop
  retrying) instead of one generic retry path.
- Stale-run recovery: a crashed worker's stuck "running" rows are reaped
  and re-enqueued automatically at the next worker startup.
- A periodic re-sync scheduler as the polling fallback for whatever a
  webhook doesn't cover.
- Manual "sync now" trigger.

### Added — Data quality & security

- Shared normalization (domain/email/company-name canonicalization) used
  at every ingestion point.
- Automatic + on-demand duplicate-company merging, reassigning contacts,
  decision makers, sponsors, and pain points to the surviving record.
- Credential encryption key rotation (`CREDENTIALS_ENCRYPTION_KEY` +
  `_PREVIOUS`, tagged ciphertext, `POST /integrations/rotate-credentials`).
- `Revoke` (wipe credentials, keep history) as a distinct action from a
  hard delete.

### Fixed

- Zapier MCP responses over bufio.Scanner's 64KB default token size
  failed every sync with "token too long" — caught against a live server
  with 119 connected apps / 310 actions.
- A killed/restarted worker process left `sync_runs` permanently stuck at
  `running`, wedging that integration's syncs forever.
- A sync task for a since-deleted integration retried indefinitely.
- The onboarding "Connect workspace" step persisted a pasted Zapier token
  in plaintext into `onboarding_states.payload`, separately from (and
  bypassing the encryption of) the real connect flow.

### Verification

Beyond unit tests, this was verified against live accounts during
development: real Apollo API key validation and enrichment, real Zapier
MCP connection (119 connected apps, 310 actions discovered from an actual
account), and real sync execution through the background worker — not
just "looks connected" in the UI. Two real bugs were caught and fixed
this way (the scanner buffer limit and the plaintext-token onboarding
leak) that no amount of pure unit testing would have surfaced on their
own. See "Testing approach" and "Known limitations" in
[backend/docs/INTEGRATIONS.md](backend/docs/INTEGRATIONS.md) for detail.
