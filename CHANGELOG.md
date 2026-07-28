# Changelog

## Integrations: Zapier, Notion, Apollo (production-grade rewrite)

This entry covers the session that took the integration layer from a
minimal connectivity check into a production-grade system: real OAuth
with refresh, real webhooks, incremental sync, automatic retry/recovery,
duplicate merging, credential rotation, and a live observability
dashboard. See [backend/docs/INTEGRATIONS.md](backend/docs/INTEGRATIONS.md)
for the full architecture writeup.
