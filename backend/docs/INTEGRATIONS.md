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
