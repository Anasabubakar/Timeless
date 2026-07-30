# Disaster Recovery Plan

## Scope
Postgres (system of record), Redis (sessions, rate-limit state,
OAuth-flow state — ephemeral by design), MinIO/S3 (uploaded files).

## Backups
- **Postgres**: automated backups are a hosting-provider setting
  (managed Postgres — RDS/Neon/Render/etc. — all offer point-in-time
  recovery). Not application-level config; confirm it's actually
  turned on for the production database as a pre-launch checklist item
  (see SECURITY_CHECKLIST.md). Recommended: daily snapshot + WAL-based
  point-in-time recovery, >= 30-day retention.
- **Redis**: treat as disposable/ephemeral. Losing it means active
  sessions' Redis-side blacklist cache is gone (the DB-backed session
  table is still authoritative for revocation — see
  SECURITY_ARCHITECTURE.md) and in-flight OAuth state/rate-limit
  counters reset. No backup needed; a cold Redis is a degraded-but-
  functional state, not a data-loss event.
- **MinIO/S3**: enable versioning and cross-region replication at the
  bucket level (hosting-provider setting). Uploaded file keys are
  content-addressed under `orgs/{orgID}/{folder}/...` — losing the
  bucket loses uploaded files, not database records (they're separate
  concerns; a `Company` record survives a lost logo image).

## Recovery procedure
1. Provision a fresh Postgres instance (or restore in place) from the
   most recent backup / point-in-time target.
2. Point `DATABASE_URL` at it, boot one instance of the API — GORM's
   `AutoMigrate` (the authoritative schema mechanism; see
   `internal/database/migrate.go`) brings the schema current on boot,
   idempotently.
3. Provision/point at a fresh Redis — no restore needed (ephemeral).
4. Verify `/health/ready` returns 200 (confirms both DB and Redis are
   actually reachable, not just that the process started).
5. Spot-check a handful of orgs' data against known-good values before
   declaring recovery complete.

## RTO / RPO targets
Not yet formally set for this project (no production deployment
exists yet to establish an SLA against). Recommended starting point
for a B2B SaaS at this stage: RPO <= 1 hour (point-in-time recovery
covers this), RTO <= 4 hours for a full region-level failure. Revisit
once real usage/customer commitments exist.

## What this plan doesn't cover
- Multi-region active-active failover (not architected for it yet —
  single-region deployment is the current assumption).
- A tested, timed recovery drill — this document describes the
  procedure; it hasn't been rehearsed against a real restore in this
  sandbox (no provisioned backup infrastructure exists here to drill
  against). Recommended before launch: actually run this procedure
  once against a staging snapshot and time it.
