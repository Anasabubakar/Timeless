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

- **Notion write-back isn't wired into every entity update path.** The
  conflict-safe primitive (`UpdatePageProperties`) and its API endpoint
  exist and are tested, but automatically pushing every Sponsor/Contact/
  Company edit back to a linked Notion page requires a persistent
  entity-to-Notion-page mapping this session didn't build — today it's a
  capability callers can use, not something that fires on every internal
  edit automatically.
- **"Webhooks by Zapier" (inbound Catch Hook receiver) isn't built.**
  Research confirmed it's the right complementary mechanism to MCP for
  event-driven flows Zapier's `tools/call` doesn't cover well, but this
  session focused on the MCP consumption path (SponsorOS calling out
  through Zapier), not receiving inbound Zap-triggered events.
- **Zapier's classic-mode app grouping is a heuristic.** Zapier doesn't
  document a tool-naming convention, so `appSlugFromAction` guesses at
  app boundaries from name prefixes. Agentic mode (when a user's server
  has it enabled) avoids this by returning real structured data instead.
