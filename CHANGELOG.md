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
