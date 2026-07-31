# Integrations Architecture

SponsorOS connects to three external systems: **Zapier** (primary gateway),
**Notion**, and **Apollo** (secondary). This document covers how the
integration layer is structured, why each design decision was made, and
what to do if something breaks.

Zapier is tried first for anything it can reach — Google Calendar, Gmail,
Slack, HubSpot, and dozens of other apps a user has already connected
through their own Zapier account — precisely because SponsorOS doesn't
need a dedicated native client for each of those apps. Notion and Apollo
get native clients because they're core to the product (workspace content
sync, prospecting data) and benefit from deeper, provider-specific
integration than a generic MCP action call can offer.

## Contents

- [Client interface](#client-interface)
- [OAuth flow (Notion)](#oauth-flow-notion)
- [Credential storage & encryption](#credential-storage--encryption)
- [Credential rotation](#credential-rotation)
- [Zapier: agentic vs. classic mode](#zapier-agentic-vs-classic-mode)
- [Zapier: safe read-only sync policy](#zapier-safe-read-only-sync-policy)
- [Notion: schema/database discovery](#notion-schemadatabase-discovery)
- [Notion: incremental sync via watermark](#notion-incremental-sync-via-watermark)
- [Notion: webhooks](#notion-webhooks)
- [Notion: conflict-safe write-back](#notion-conflict-safe-write-back)
- [Apollo: organization enrichment](#apollo-organization-enrichment)
- [Apollo: role-based contact discovery](#apollo-role-based-contact-discovery)
- [Data quality: normalization](#data-quality-normalization)
- [Data quality: dedupe/merge](#data-quality-dedupemerge)
- [Background workers: retry/backoff](#background-workers-retrybackoff)
- [Background workers: stale-run recovery](#background-workers-stale-run-recovery)
- [Background workers: periodic re-sync scheduler](#background-workers-periodic-re-sync-scheduler)
- [Observability: sync_runs table](#observability-sync_runs-table)
- [Observability: API endpoints](#observability-api-endpoints)
- [Environment variables](#environment-variables)
- [Extending: adding a new provider](#extending-adding-a-new-provider)
- [Testing approach](#testing-approach)
- [Troubleshooting](#troubleshooting)
- [Known limitations](#known-limitations)

## Client interface

Every provider implements `integration.Client` (`internal/integration/client.go`):

```go
type Client interface {
    Provider() string
    Validate(ctx context.Context, credentials map[string]string) error
    Sync(ctx context.Context, credentials map[string]string, state map[string]interface{}) (*SyncResult, error)
}
```

`Validate` must make a real call to the provider — it's what "immediately
validate the key" means in practice, not a format check. `Sync` takes the
previous run's `state` (cursors/watermarks) and returns new `state` to
persist, so a provider can resume incrementally instead of re-processing
everything every run.

Providers whose tokens can expire and be silently renewed also implement
`Refresher`:

```go
type Refresher interface {
    Refresh(ctx context.Context, credentials map[string]string) (map[string]string, error)
}
```

## OAuth flow (Notion)

Notion is the only provider with a real customer-facing OAuth app (Apollo
and Zapier don't offer OAuth to regular API-key customers — see their
sections below). The flow:

1. `GET /integrations/notion/oauth/start?token=<jwt>` — the frontend can't
   attach an Authorization header to a top-level browser navigation, so the
   session JWT is passed as a query param instead. A random `state` value
   is minted and stored in Redis for 10 minutes, keyed to the org/user.
2. Notion redirects to `GET /integrations/oauth/callback?code=...&state=...`.
   The state is looked up (and deleted) from Redis, the code is exchanged
   for a token via HTTP Basic auth against `POST /v1/oauth/token`, and the
   resulting credentials are handed to `IntegrationService.Connect`.
3. Notion's token response includes `access_token`, `refresh_token`,
   `workspace_id`, `workspace_name`, `workspace_icon`, and `bot_id` — all of
   which are preserved in the encrypted credentials blob.

## Credential storage & encryption

Credentials are never stored in the clear. `IntegrationService` encrypts
the credentials map (JSON-marshaled, then AES-256-GCM-sealed) before
writing to `Integration.Credentials`, and decrypts on the way out — no
handler or frontend code ever sees a raw token. See
`internal/security/crypto.go` for the cipher implementation.

## Credential rotation

`CredentialCipher` tags every ciphertext with the id of the key it was
encrypted under. Rotating `CREDENTIALS_ENCRYPTION_KEY`:

1. Move the current value into `CREDENTIALS_ENCRYPTION_KEY_PREVIOUS`
   (comma-separated list, oldest first).
2. Set a new `CREDENTIALS_ENCRYPTION_KEY`.
3. Redeploy, then call `POST /integrations/rotate-credentials` for each
   org (or loop over all orgs from an admin script) to re-encrypt every
   stored credential under the new key. Rows already on the current key
   are skipped, so the call is safe to repeat.

Skipping step 3 doesn't break anything immediately — `Decrypt` still finds
the old key in the previous-keys list — but the org stays on a retired key
until rotation is completed.

## Zapier: agentic vs. classic mode

A user's Zapier MCP server (`mcp.zapier.com`) runs in one of two modes:

- **Classic mode**: every enabled action is its own MCP tool, discoverable
  only via `tools/list`. `groupToolsByApp` infers the source app from each
  tool name's prefix (a heuristic — Zapier doesn't document a slug format).
- **Agentic mode (beta)**: a small fixed set of meta-tools —
  `list_enabled_zapier_actions`, `discover_zapier_actions`,
  `execute_zapier_read_action`, `execute_zapier_write_action` — give real
  structured discovery instead of name-parsing.

`ZapierClient.DiscoverApps` tries agentic mode first and falls back to
classic-mode grouping automatically; `SyncResult.Details["mode"]` records
which one was actually used.

## Zapier: safe read-only sync policy

A sync pass never blindly invokes every discovered tool — some Zapier
actions send emails, post messages, or delete data. `isSafeReadOnlyTool`
only allows a tool to be auto-invoked during sync if its name matches a
read verb (`list_`, `search_`, `get_`, `find_`, `fetch_`), doesn't contain
a write verb (`send`, `create`, `delete`, `update`, `post`, ...), and has
zero required input arguments. Anything else is left alone — it's only
ever invoked explicitly via `ExecuteAction` when a calling service
deliberately wants that specific action.

## Notion: schema/database discovery

`NotionClient.Sync` paginates `/v1/search` (sorted newest-edited-first),
and for every database result reads its `data_sources` (the 2025-09-03 API
split a database into a container plus one or more data sources) via
`/v1/data_sources/{id}`. No database or property name is ever hardcoded —
`harvestDataSourceRows` inspects each data source's schema at query time to
decide whether it looks like a contacts table (a title property plus an
email/company-like column) and maps rows into `ContactRecord` accordingly,
falling back to `NoteRecord` for anything else so nothing is silently
dropped.

## Notion: incremental sync via watermark

`discoverAll` stops paginating as soon as it crosses the stored watermark
(the newest `last_edited_time` seen on the previous run) — a steady-state
re-sync only reads what actually changed instead of re-walking the entire
workspace. The watermark round-trips through `SyncResult.State` and
`Integration.Config` (see `mergeConfig`/`extractState` in the worker
package), not through any Notion-side cursor, since Notion's search API
doesn't expose one.

## Notion: webhooks

`POST /integrations/notion/webhook` handles Notion's real-time events
(`developers.notion.com/reference/webhooks`):

1. **Verification handshake**: the first request after saving this URL in
   the integration's Webhooks configuration tab carries
   `{"verification_token": "..."}`. It's stored in Redis
   (`notion:webhook:verification_token`) — this is app-wide (one Notion
   OAuth app, one webhook endpoint), not per-org.
2. **Signed events**: every subsequent request is verified via
   `X-Notion-Signature: sha256=<hex>` — HMAC-SHA256 over the raw body,
   keyed by the stored verification token. An invalid signature is
   rejected with 401 before the payload is even parsed.
3. Verified events are matched to an org by `workspace_id` (via
   `Integration.ExternalAccountID`, populated at connect time) and
   dispatched as an incremental sync job — the HTTP handler itself does no
   processing, so a burst of Notion activity can't make the endpoint slow.

## Notion: conflict-safe write-back

`NotionClient.UpdatePageProperties` re-reads a page's current
`last_edited_time` immediately before writing. If it's newer than the
`expectedLastEditedTime` the caller last read, the write is refused with
`ConflictError` rather than silently overwriting — the concrete mechanism
behind "never overwrite newer data with stale data." Exposed via
`PATCH /integrations/notion/pages/:pageID`, which maps a conflict to HTTP
409 so the frontend can show a specific "this changed in Notion" message
instead of a generic failure.

## Bidirectional sync: event bus, mapping engine, sync ledger

The pieces above (OAuth, credential encryption, one-off conflict-safe
write-back) predate a second, event-driven sync layer that now sits on
top of them. This is the part that makes Notion write-back automatic
(instead of a capability callers had to invoke manually) and adds real
inbound Zapier support.

**`internal/eventbus`** — an in-process pub/sub `Bus`. Entity services
(`CompanyService`, `ContactService`, `SponsorService`) call
`bus.Publish(...)` on every create/update/delete via a small
`SetBus`/`publish` helper each one has; this is intentionally *not* part
of their constructor signature, so nothing outside `router.Setup` (and
`cmd/worker/main.go`, for the worker's own subscribers) is forced to wire
a bus through just to construct one of these services in a test.
`Bus.Publish` routes through `worker.NewEventPublisher` (an asynq
enqueue) when `SetPublisher` has been called — so an event survives a
process restart the same way any other background job does — and falls
back to synchronous in-process dispatch otherwise (what unit tests get by
default).

**`internal/mapping`** — the `Adapter` interface (`ToExternal`/
`FromExternal`/`Push`/`Fetch`/`Archive`) plus a Notion implementation
(`NotionAdapter`), driven entirely by a per-org, user-configured
`models.FieldMapping` (org + integration + entity type + external
container id + a JSON array of `{internal_field, external_field,
external_type, direction}` entries) — never a hardcoded database/property
ID. `internal/mapping/extract.go` converts real models
(`CompanyToRecord`/`ContactToRecord`/`SponsorToRecord`) into the generic
`SyncableRecord` shape adapters work with, and back
(`ApplyToCompany`/`ApplyToContact`/`ApplyToSponsor`) for the inbound path.

**`internal/syncengine`** — the two subscribers that actually do
something with events:
- `PushService` subscribes to every Company/Contact/Sponsor CRUD event,
  looks up active `FieldMapping`s for that org+entity type, and pushes the
  *current* record (re-fetched fresh, not the event's payload) through the
  matching adapter — creating the `SyncedEntity` ledger row on first sync,
  updating it after. A delete event archives the external record instead
  of pushing empty fields.
- `PullService` reacts to `NotionChanged` (published by the webhook
  receiver — see below) by fetching the changed page and reconciling it
  against whichever internal entity the `SyncedEntity` ledger says it's
  linked to. It applies the change locally *unless* the local record was
  also modified since the last sync, in which case it's marked
  `sync_state = conflict` instead of guessing a winner. Applying a pulled
  change writes straight through the repository (not the entity service),
  specifically so it never re-publishes a CRUD event and ping-pongs the
  same change back out to Notion.
- `ZapierIngestService` subscribes to `ZapierWebhookReceived` (see below)
  and turns a recognized `{"event_type": "contact"|"lead", ...}` payload
  into a real `Contact`, created *through* `ContactService` — which means
  it publishes `ContactCreated` like any other creation path, and
  `PushService` picks it up and syncs it to Notion with no Zapier-specific
  code in the Notion adapter at all.

**`models.SyncedEntity`** is the per-record ledger this all keys off:
`(organization_id, entity_type, entity_id, external_system)` uniquely
identifies one row, tracking `sync_state` (`pending`/`synced`/`conflict`/
`error`), a logical `version` counter, which side (`source`) produced the
most recent change, and `last_modified_local`/`last_modified_remote`/
`last_synced_at` timestamps for conflict detection. `models.SyncHistory`
is the append-only action log (`pushed_to_remote`, `pulled_from_remote`,
`conflict_detected`, `conflict_resolved`, `sync_failed`) per ledger row.

## Zapier: inbound webhooks

Unlike Notion, "Webhooks by Zapier" (the inbound trigger a user's Zap
posts to) has **no signing mechanism at all** — there's no HMAC signature
to verify, no shared-secret header Zapier itself sends. The entire
authentication is an unguessable per-org URL token:

1. `POST /integrations/:provider/webhook-token` (authenticated,
   `integrations:write`) generates a random 32-byte token on first call
   and stores it in `Integration.WebhookSecret` — the same field Notion's
   `WebhookURL`/`WebhookSecret` pair already existed for, repurposed here
   as "the URL segment IS the secret" rather than "a value used to verify
   someone else's signature." The response (`webhook_url`) should be
   treated like a credential — it's the org's Zap URL to paste into
   "Webhooks by Zapier," and it's never re-derivable from the normal
   integration listing.
2. `POST /webhooks/zapier/:token` (public) resolves the token to an
   `active` Integration via `IntegrationByWebhookToken`, rejects anything
   else with an identical 401 response/timing profile (never a 404 that
   would let an attacker distinguish "unknown token" from other failure
   modes), content-hash-dedupes the payload in Redis for 24h (Zapier
   retries on anything but a 2xx, so a redelivered duplicate is
   acknowledged without reprocessing), and publishes
   `ZapierWebhookReceived` with the raw JSON body as `Data` — no inline
   processing, so a slow/failing downstream subscriber never becomes a
   slow/failing response to Zapier.

The one payload shape handled out of the box is a new contact/lead (see
`ZapierIngestService` above); anything else is left alone rather than
guessed at, but the raw event is still durably queued for a future
processor to pick up.

## Apollo: organization enrichment

`ApolloClient.EnrichOrganization` calls `GET /api/v1/organizations/enrich`
by domain and returns firmographic data (industry, employee count,
revenue, funding, technologies, location) as-is from Apollo — the worker
merges it directly into `Company.EnrichmentData`. Returns `(nil, nil)` —
not an error — when Apollo simply has no record for a domain, so "no
data" is never treated as a sync failure.

## Apollo: role-based contact discovery

`ApolloClient.DiscoverRoleContacts` searches `TargetRoles` (Founder, CEO,
Co-Founder, CMO, Marketing Director, Head of Partnerships, Partnership
Manager, Developer Relations, Community Lead, Brand Lead, Country
Manager, Regional Manager, Events Manager, Communications Lead) against a
company's domain via `mixed_people/api_search`, and reports exactly one
`DiscoveredContactRecord` per role — `Available: false` (never a
fabricated name/email) when nobody matches. Email reveals are
credit-consuming, so they're capped at `maxEmailRevealsPerCompany` (6) per
company per run, prioritized in `TargetRoles` order; confidence is derived
from Apollo's own `email_status` (`verified` → 0.95, `guessed` → 0.4,
`unverified` → 0.3), never asserted independently of it.

## Data quality: normalization

`internal/normalize` centralizes canonicalization: `Domain` strips
scheme/`www.`/path/port so `Example.com`, `www.example.com`, and
`https://example.com/` all compare equal; `Email` lowercases and trims;
`CompanyName` strips common legal suffixes (Inc, LLC, Ltd, Corp, ...) for
comparison purposes only — the original human-entered name is always what
gets displayed/stored. Every ingestion path normalizes before it looks up
an existing row, which is what actually prevents the duplicate in the
first place (as opposed to merging it after the fact).

## Data quality: dedupe/merge

`internal/dedupe.MergeDuplicateCompanies` groups an org's companies by
normalized domain (falling back to normalized name), keeps the most
complete record in each group as primary (`completenessScore` counts
filled-in fields), reassigns every related row (contacts, decision makers,
sponsors, pain points) to it, unions tags, and soft-deletes the rest. It
runs automatically after any sync that ingested contacts, and is also
exposed as an on-demand maintenance action via `POST /companies/dedupe`.
It lives in its own package (not under `service`, which imports `worker`)
specifically so the background worker can call it too without an import
cycle.

## Background workers: retry/backoff

Sync failures aren't all handled the same way:

- **Rate limit** (`integration.RateLimitError`): status → `retrying`, and a
  custom asynq `RetryDelayFunc` honors the provider's own `Retry-After`
  header instead of asynq's generic exponential curve.
- **Auth expired** (`integration.AuthExpiredError`): the runner tries the
  provider's `Refresher` (if it implements one) and retries once with the
  rotated credentials. If that fails too, status → `expired` and the job
  is told to stop retrying (`asynq.SkipRetry`) — retrying with the same
  stale token can't ever succeed.
- **Integration deleted mid-flight**: also `asynq.SkipRetry` — the record
  can't reappear.
- Anything else: status → `error`, and asynq's default retry/backoff
  applies.

## Background workers: stale-run recovery

If a worker process is killed mid-sync, the `sync_runs` row it was writing
to is left stuck at `status: "running"` forever — which would otherwise
permanently block that integration from ever syncing again, since
`HasRunning` is the guard against duplicate concurrent syncs.
`SyncRunRepository.HasRunning` only counts a `running` row younger than
`staleRunThreshold` (10 minutes); `ReapStaleRuns` marks anything older as
`failed`, and `worker.RecoverStaleSyncs` (run once at worker startup)
calls it and re-enqueues a fresh sync for whatever integration was left
stuck — recovery without a human needing to notice.

## Background workers: periodic re-sync scheduler

`worker.StartPeriodicResync` ticks every 5 minutes and enqueues a sync
(trigger `scheduled`) for any `active` integration whose `last_sync_at` is
older than `resyncInterval` (15 minutes) and isn't already mid-sync — the
polling fallback for whatever a webhook doesn't cover (Apollo/Zapier have
no event-push mechanism at all; Notion's webhooks cover most but not
necessarily every change type).

## Observability: sync_runs table

Every sync execution — connect, scheduled, webhook-triggered, or manual —
is recorded as a `models.SyncRun` row: trigger, status, started/finished
timestamps, duration, records synced, warnings, and error. This is what
lets the dashboard show real history ("2 failed syncs in the last 24h",
per-run duration and record counts) instead of just a single
`last_sync_at` timestamp.

## Observability: API endpoints

| Endpoint | Purpose |
|---|---|
| `GET /integrations/dashboard` | Every integration's health, recent sync runs, and 24h failure count |
| `GET /integrations/zapier/apps` | Connected apps discovered through Zapier |
| `POST /integrations/:id/sync` | Manual "sync now" |
| `POST /integrations/:id/revoke` | Wipe credentials, keep history |
| `POST /integrations/rotate-credentials` | Re-encrypt credentials still on a retired key |
| `POST /companies/dedupe` | On-demand duplicate-company merge |
| `PATCH /integrations/notion/pages/:pageID` | Conflict-safe write-back to a Notion page |
| `POST /integrations/notion/webhook` | Notion's real-time event receiver (public, signature-verified) |
| `GET /integrations/sync/conflicts` | Every `SyncedEntity` awaiting conflict resolution, org-wide |
| `GET /integrations/sync/activity` | Recent `SyncHistory` entries (pushed/pulled/conflict/failed), org-wide |
| `POST /integrations/:provider/webhook-token` | Generate (or fetch, if already generated) an inbound webhook URL for a provider with no signing scheme of its own (Zapier) |
| `POST /webhooks/zapier/:token` | Zapier's inbound event receiver (public, token-in-path authenticated) |

## Environment variables

| Variable | Purpose |
|---|---|
| `NOTION_CLIENT_ID` / `NOTION_CLIENT_SECRET` | Notion OAuth app credentials |
| `CREDENTIALS_ENCRYPTION_KEY` | Dedicated secret for encrypting stored credentials (falls back to `JWT_SECRET`) |
| `CREDENTIALS_ENCRYPTION_KEY_PREVIOUS` | Comma-separated retired encryption secrets, for key rotation |
| `API_PUBLIC_URL` | Used to build the Notion OAuth redirect URI |
| `FRONTEND_URL` | Where OAuth callbacks redirect back to after connect |

Apollo and Zapier have no OAuth env vars — Apollo is API-key-only for
regular customers, and Zapier has no third-party OAuth app registration
(the user pastes a personal MCP connection token instead).

## Extending: adding a new provider

1. Implement `integration.Client` (and `Refresher` if it has expiring
   OAuth tokens) in `internal/integration/<provider>.go`.
2. Register it in `integration.Registry`.
3. If it needs OAuth, add its client id/secret to `config.Config` and
   register an `OAuthProvider` entry in `handler.NewOAuthHandler`.
4. If it needs a webhook receiver, follow the Notion pattern: verify a
   signature before trusting the payload, route by an `ExternalAccountID`
   equivalent, and only ever enqueue a job from the handler — never
   process inline.
5. Add its provider string to the frontend's `AVAILABLE_PROVIDERS` list
   once it actually has a working client — never add a placeholder entry
   for a provider with no real backend support behind it.

## Testing approach

Pure logic (slug guessing, role matching, state merging, normalization,
dedupe-key derivation) is covered by ordinary table-driven unit tests with
no external dependencies. HTTP-facing logic (status-code-to-error-type
mapping, the event-stream scanner) is covered with `httptest.Server`
instead of mocks, so the actual `net/http` request/response path is
exercised — this is what caught the 64KB scanner buffer bug in the first
place, by re-running the exact same code against a live Zapier MCP server
before it was caught in a test.

Full end-to-end connect → sync → dashboard flows were also verified
against live Notion/Apollo/Zapier accounts during development (not just
unit tests) — see the "Known limitations" section below for what that
verification could and couldn't cover in a sandboxed environment.

**Bidirectional sync layer specifically** (`internal/eventbus`,
`internal/mapping`, `internal/syncengine`): `eventbus.Bus`'s pub/sub fan
out/aggregate-error behavior, the Notion property (de)serializers,
`FieldMapping` direction filtering, entity→`SyncableRecord` extraction
and the inverse `ApplyTo*` merge (only touches fields present in the
incoming map, never clobbers with a blank value), and
`ZapierIngestService`'s early-return guards (unrecognized event type,
no identifying fields, invalid org id) are all covered by ordinary unit
tests with no external dependencies — see `internal/eventbus/
eventbus_test.go`, `internal/mapping/{mapping,notion,extract}_test.go`,
`internal/security/crypto_test.go` (the shared stored-credentials
encrypt/decrypt helper), and `internal/syncengine/zapier_ingest_test.go`.

What's **not** covered by automated tests in this environment:
`PushService`/`PullService`'s full DB-backed flows (find-or-create the
`SyncedEntity` ledger row, apply a pulled change, detect a conflict from
real `last_modified_local`/`_remote` timestamps) and
`ZapierWebhookHandler`'s token lookup + Redis dedupe, because both
require a real Postgres and Redis instance this sandbox doesn't have
network access to provision (no `miniredis`/sqlite-in-memory harness
exists in this codebase yet, and this sandbox has no route to
proxy.golang.org to add one). The logic itself was still written
defensively and reviewed by hand (see the package doc comments in
`internal/syncengine`), but "does the SQL actually behave this way
against a real Postgres" and "does SETNX actually dedupe across two
near-simultaneous requests" are exactly the kind of thing that should be
verified against a real staging environment before launch — see the
manual verification checklist below.

### Manual verification checklist (needs a live environment)

Run this against a staging deployment with real Postgres, Redis, and a
real Notion workspace + Zapier account connected, before relying on
bidirectional sync in production:

1. **Notion push**: create a `FieldMapping` for `company` against a real
   Notion database, create a Company in Timeless, confirm a new page
   appears in that database with the mapped properties populated, and a
   `SyncedEntity` row exists with `sync_state = synced`.
2. **Notion push, update**: edit that Company in Timeless, confirm the
   Notion page's properties update and `SyncedEntity.version` increments.
3. **Notion push, delete**: delete the Company, confirm the Notion page
   is archived (not hard-deleted).
4. **Notion pull**: edit the property directly in Notion, confirm the
   webhook fires, `NotionChanged` is published, and the Timeless record
   updates to match — without re-triggering another outbound push (watch
   `sync_history` for a `pulled_from_remote` entry, not a ping-ponging
   pair of `pushed_to_remote`/`pulled_from_remote` entries).
5. **Conflict**: edit the same record in both Timeless and Notion within
   the same window without letting either sync first, confirm the ledger
   row lands in `sync_state = conflict` rather than either side silently
   winning.
6. **Zapier inbound**: generate a webhook token via
   `POST /integrations/zapier/webhook-token`, wire a real Zap's
   "Webhooks by Zapier" action to POST
   `{"event_type": "contact", "email": "...", "first_name": "...", ...}`
   to it, confirm a Contact is created and — if a Notion `FieldMapping`
   for `contact` exists — that it also appears in Notion, proving the
   whole Zapier → Timeless → Notion chain works with zero
   Zapier-specific code in the Notion adapter.
7. **Zapier duplicate delivery**: replay the same Zapier payload (Zapier
   itself will do this on any non-2xx response, or trigger it manually)
   and confirm no duplicate Contact is created.
8. **Zapier invalid token**: POST to `/webhooks/zapier/some-made-up-token`
   and confirm a 401 with no information leakage about which tokens are
   valid.

## Troubleshooting

- **"oauth_not_configured" on connect**: `NOTION_CLIENT_ID`/`_SECRET` (or
  equivalent) aren't set — the frontend surfaces this specific error
  rather than a generic failure so it's clear which env var is missing.
- **Integration stuck on "Syncing..." indefinitely**: check
  `GET /integrations/dashboard` for the integration's `recent_runs` — a
  `failed` run with a real error message will be there. If the worker
  process was killed mid-sync, `RecoverStaleSyncs` fixes this
  automatically the next time the worker starts (within
  `staleRunThreshold`, 10 minutes).
- **"reconnect required" / status `expired`**: the stored token was
  rejected as expired/revoked and (if the provider supports it) a refresh
  attempt also failed — the user needs to go through Connect again.
- **Rate limited**: status shows `retrying`; the sync_run's `error` field
  contains the provider's own Retry-After value, and the worker will
  retry at that exact time without any manual action needed.

## Known limitations

- **Notion write-back is now automatic for Company/Contact/Sponsor**,
  driven by the event bus + mapping engine (see "Bidirectional sync"
  above) — the earlier limitation here (write-back was a capability
  callers had to invoke manually) is resolved. It does not yet cover
  every entity type in the product (Meeting, Task, Project, Note,
  Proposal, Campaign) — extending `entityLoaders`/`entityStates` in
  `internal/syncengine` and the corresponding `*ToRecord`/`ApplyTo*`
  functions in `internal/mapping/extract.go` is additive, not a redesign,
  when those are needed.
- **Inbound Zapier webhooks are now built** (see "Zapier: inbound
  webhooks" above), resolving the earlier limitation here. Only one
  payload shape is handled by default (new contact/lead) — a Zap sending
  any other `event_type` is durably queued as `ZapierWebhookReceived` but
  has no subscriber yet; add one the same way `ZapierIngestService` is
  wired in `cmd/worker/main.go` when a second shape is needed.
- **Zapier's classic-mode app grouping is a heuristic.** Zapier doesn't
  document a tool-naming convention, so `appSlugFromAction` guesses at
  app boundaries from name prefixes. Agentic mode (when a user's server
  has it enabled) avoids this by returning real structured data instead.
- **No automated DB/Redis integration tests for the sync layer** in this
  environment — see "Testing approach" above for exactly what is and
  isn't covered, and the manual verification checklist for what to run
  against a real staging environment before launch.
- **Conflict resolution has no resolve-it-for-me UI/API yet.** A
  conflicted `SyncedEntity` is surfaced (dashboard's conflict queue,
  `GET /integrations/sync/conflicts`) but there's no endpoint to record a
  decision (`ConflictResolution`/`ConflictDetails` exist on the model,
  written by nothing yet) — today a human resolves it by editing
  whichever side should win and waiting for the next sync to pick it up
  cleanly (the conflict clears the next time push/pull succeeds without
  detecting another conflict).

---

See [CHANGELOG.md](../../CHANGELOG.md) at the repo root for what shipped
in the session that produced this document.
