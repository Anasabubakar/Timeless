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
