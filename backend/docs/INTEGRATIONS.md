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
