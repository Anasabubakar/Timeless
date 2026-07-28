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
