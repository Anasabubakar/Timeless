# Dependency Audit

Snapshot taken 2026-07-30. Live vulnerability-database lookups
(`govulncheck`, `npm audit`) were unreliable in this sandbox — the
Go module proxy and npm's audit endpoint were both intermittently
unreachable — so this is a manual review against `go.mod`/
`package-lock.json` plus targeted searches for known CVEs, not a
substitute for running both tools in CI. See "Follow-up" below.

## Backend (Go)

All direct dependencies in `go.mod` are on recent, actively maintained
versions (`golang-jwt/jwt/v5` v5.2.1, `gorm.io/gorm` v1.25.12,
`redis/go-redis/v9` v9.7.0, `hibiken/asynq` v0.25.1,
`go-playground/validator/v10` v10.27.0, `golang.org/x/crypto` v0.33.0)
with no known critical/high CVEs at time of writing. Two findings:

1. **`github.com/gofiber/fiber/v3` is pinned to `v3.0.0-beta.4`.**
   Fiber v3 has since reached a stable `v3.0.0` and progressed to
   `v3.3.0`, several releases past the beta this app runs. Beta
   releases predate the stability/security hardening that goes into a
   1.0 cut, and Fiber's own docs mention a migration CLI
   (`fiber migrate --to v3.0.0`) for the beta→stable jump — implying
   breaking changes, not just a version bump. **Not upgraded as part of
   this pass**: this sandbox has no live Postgres/Redis to run the app
   against, so a major-version framework bump can't be verified beyond
   `go build`/`go vet`/the existing test suite, and this specific
   upgrade needs full route-by-route regression testing. Recommended
   follow-up: run `fiber migrate --to v3.0.0`, then `v3.3.0`, in a
   branch with a real environment available, and exercise every route
   group manually before merging.

2. **`github.com/lib/pq` is in maintenance mode** (no new features,
   only critical fixes). It's used directly for its `pq.StringArray`
   GORM-compatible type (Company/Contact tag columns) — not for the
   database connection itself, which already goes through
   `gorm.io/driver/postgres`. Low priority: this is a "legacy but
   stable" hygiene item, not an active vulnerability. Migrating off it
   means replacing `pq.StringArray` usage in
   `internal/models/company.go`, `internal/models/contact.go`,
   `internal/handler/company.go`, `internal/handler/contact.go`, and
   `internal/dedupe/dedupe.go` with a `pgtype`-based equivalent — a
   real (if small) data-layer change that needs a live database to
   verify array (de)serialization doesn't regress, so left as a
   follow-up rather than done blind in this session.

No other backend dependency stood out as outdated, abandoned, or
carrying an unpatched known CVE at the versions currently pinned.

## Frontend (Next.js)

`package.json` pins floors (`next": "^15.1.0"`, `"react": "^19.0.0"`)
but the actual installed versions per `package-lock.json` are
`next@15.5.21` and `react@19.2.8` — both current, both well past the
first Next.js middleware-authorization-bypass CVE's patched threshold
(CVE-2025-29927, fixed in 15.2.3+).

Checked specifically because 2026 saw two further Next.js
middleware-authorization-bypass CVEs (CVE-2026-44574, CVE-2026-44575)
affecting App Router deployments that gate access through
`middleware.ts`: **this app has no `middleware.ts` at all**, and no
`src/app/api/*` route handlers either (verified — neither file/directory
exists in `frontend/`). The frontend is a pure client of the Go
backend API; every authorization decision already happens server-side
in the Go backend (RouteGuard/RBAC, hardened in an earlier phase), not
in Next.js middleware. So while confirming the installed version is
fine either way, these three CVEs specifically don't apply to this
app's architecture regardless of Next.js patch level, because there's
no middleware-based auth gate for them to bypass.

## Follow-up (needs live tooling this sandbox didn't reliably have)

- Wire `govulncheck ./...` (backend) and `npm audit --omit=dev`
  (frontend) into CI so every PR gets a live vulnerability-database
  check instead of the manual snapshot above.
- Revisit the Fiber v3 beta→stable upgrade in an environment with a
  real Postgres/Redis to test against.
