# Changelog

## Bidirectional sync: event bus, mapping engine, Zapier inbound webhooks

This entry covers turning the integration layer from one-directional
(Timeless reads from Notion/Apollo/Zapier) into a true bidirectional,
event-driven sync system. See "Bidirectional sync" and "Zapier: inbound
webhooks" in [backend/docs/INTEGRATIONS.md](backend/docs/INTEGRATIONS.md)
for the full architecture writeup.

### Added — Event bus & mapping engine

- `internal/eventbus.Bus`: in-process pub/sub, durable via the existing
  asynq worker queue (`SetPublisher`) so an event survives a process
  restart. `CompanyService`/`ContactService`/`SponsorService` publish
  Created/Updated/Deleted events on every write.
- `internal/mapping`: a generic `Adapter` interface
  (`ToExternal`/`FromExternal`/`Push`/`Fetch`/`Archive`) plus a Notion
  implementation, driven entirely by a per-org, user-configured
  `FieldMapping` — no hardcoded database/property IDs, so this works
  against whatever Notion database a user actually has.
- `models.SyncedEntity` / `models.SyncHistory`: the per-record sync
  ledger (sync state, logical version, which side changed most recently,
  conflict state) and its append-only action log.

### Added — Notion write-back and inbound sync

- `PushService`: subscribes to every Company/Contact/Sponsor CRUD event
  and pushes the current record to Notion through the mapping engine —
  write-back is now automatic, not a capability callers had to invoke
  manually.
- `PullService`: reconciles a changed Notion page against its linked
  internal entity, applying the change locally unless the local side also
  changed since the last sync — in which case it's flagged
  `sync_state = conflict` instead of guessing a winner. Applies changes
  by writing straight to the repository, specifically so an inbound pull
  never re-triggers an outbound push and ping-pongs the same change back
  and forth.
- The existing Notion webhook receiver now also publishes a targeted
  `NotionChanged` event when it can identify the changed page, so
  `PullService` runs immediately instead of waiting for the next
  scheduled resync.

### Added — Zapier inbound webhooks

- `POST /webhooks/zapier/:token`: Zapier has no signing mechanism for its
  inbound "Webhooks by Zapier" trigger, so an unguessable per-org URL
  token (generated via `POST /integrations/:provider/webhook-token`) is
  the entire authentication. Content-hash deduped in Redis for 24h
  (Zapier retries on anything but 2xx); durably publishes
  `ZapierWebhookReceived` rather than processing inline.
- `ZapierIngestService`: turns a recognized `{"event_type": "contact", ...}`
  payload into a real Contact, created through `ContactService` — which
  means `PushService` automatically syncs it out to Notion too, with zero
  Zapier-specific code anywhere in the Notion adapter.

### Added — Sync Dashboard

- `GET /integrations/sync/conflicts` and `GET /integrations/sync/activity`
  — the conflict queue and recent sync-activity feed, org-wide.
- `GET /integrations/dashboard` now also returns per-integration
  synced/pending/conflict/error counts and the last-webhook-received
  timestamp.
- A "Sync health" section on the Integrations page rendering all of the
  above.

### Fixed (found while building this)

- `mapping.Adapter.Push`'s doc comment promised it returned the external
  id on create; the signature didn't actually return one — a caller
  creating a brand-new external record had no way to learn the id Notion
  assigned it. Fixed before anything depended on the wrong signature.
- `middleware.routeguard`'s route-permission table was compiled from a Go
  map with no defined iteration order — when two patterns could both
  match the same path (a literal route and a `:param` wildcard), which
  one `lookupPermission` used was nondeterministic between process
  restarts. Every existing occurrence happened to require the same
  permission on both sides, so this was latent, not yet an actual
  privilege bug — fixed with a deterministic sort (literal patterns
  checked before wildcards) before a future route with differing
  permissions could turn it into one.

### Known limitations

Only Company/Contact/Sponsor are wired into the sync pipeline (Meeting,
Task, Project, Note, Proposal, Campaign aren't yet — extending this is
additive); only one Zapier payload shape (new contact/lead) has a
processor; there's no automated DB/Redis integration test coverage for
the sync layer in this environment (no network access to add a
Postgres/Redis test harness); conflict resolution has no
resolve-it-for-me UI yet, only surfacing. Full detail, including a manual
verification checklist for a real staging environment, in
backend/docs/INTEGRATIONS.md.

### Deployment notes

No manual database migration is required — `AutoMigrate` creates
`synced_entities`, `sync_history`, and `field_mappings` automatically, and
adds the `webhook_secret` index on `integrations`. Restart both the API
server and the worker process; existing integrations keep working
unchanged until a `FieldMapping` is actually configured for them (the
sync pipeline is a no-op per org until then).

---

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

### Known limitations

Notion write-back is a capability, not yet an automatic trigger on every
entity edit; inbound "Webhooks by Zapier" (Catch Hook) receiving isn't
built, only the MCP consumption path; Zapier's classic-mode app grouping
is a documented heuristic, not a Zapier-guaranteed contract. Full detail
in backend/docs/INTEGRATIONS.md.

### Deployment notes

No manual database migration is required — `AutoMigrate` (run on every
boot) creates the new `sync_runs` table and `integrations.external_account_id`
column automatically. Restart both the API server and the worker process
to pick up the new code; existing connected integrations keep working
unchanged (their next sync just gains history tracking and the new
retry/recovery behavior).
