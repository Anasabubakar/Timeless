# Injection / XSS / CSRF Review

Code-audit findings (this sandbox has no live Postgres/Redis to run
fuzzing or a real browser against, so this is a static review of every
relevant code path rather than a dynamic test run — see
`DEPENDENCY_AUDIT.md` for the same caveat applied elsewhere).

## SQL / ORM injection
**Reviewed: every repository query.** No raw SQL string concatenation
was found anywhere (`grep`'d for `fmt.Sprintf` near `.Raw(`/`.Exec(`
across the whole `internal/repository` package — zero matches). The
two free-text search paths (`ContactRepository.List`,
`CompanyRepository.List`) use GORM's `?` placeholder binding —
`Where("name ILIKE ? OR domain ILIKE ?", "%"+search+"%", ...)` — the
`%` wildcards are part of the *bound parameter value*, not the SQL
template, so a search term containing `'`, `;`, or SQL keywords is
passed to Postgres as an inert string literal, not parsed as SQL.
`internal/handler/batch.go`'s field-whitelist (`allowedBatchFields`)
additionally prevents a client from naming an arbitrary column via the
batch-update `fields` map.

## Cross-site scripting (XSS)
**Reviewed: frontend rendering + backend output.**
- Frontend: zero uses of `dangerouslySetInnerHTML` anywhere in
  `frontend/src` (grep confirmed). `<ReactMarkdown>` (used for AI
  responses) has no `rehypePlugins`/`allowDangerousHtml` configured, so
  it escapes raw HTML by default rather than rendering it — AI output
  can't inject a `<script>` tag through it.
- Backend: `Content-Security-Policy` and `X-Content-Type-Options:
  nosniff` are set on every response (security headers middleware),
  defense-in-depth against any XSS vector that did slip through.
- AI system prompts: `learned_context` (built from prior user-submitted
  preferences/feedback/queries) is wrapped in an explicit
  `<untrusted_learned_context>` delimiter rather than concatenated as
  trusted instructions — closes a stored-prompt-injection angle that
  isn't classic XSS but is the same "untrusted content rendered/
  interpreted as something more privileged" family of bug.

## CSRF
**Architecturally out of scope for the primary threat, verified.**
This API uses `Authorization: Bearer <jwt>` for every authenticated
request — zero `Set-Cookie`/`c.Cookie()` usage anywhere in the backend
(grep confirmed across `internal/handler` and `internal/middleware`).
Classic CSRF exploits *ambient* credentials a browser attaches
automatically (cookies); a bearer token in a header isn't attached by
the browser to a cross-origin form submission or image tag, so the
standard CSRF attack shape doesn't apply here. `ValidateOrigin` on the
protected route group and `ValidateOriginAlways` on the WebSocket
upgrade are still enforced as defense-in-depth (a WebSocket upgrade in
particular isn't subject to the Same-Origin Policy the way a normal
`fetch()` is, so Origin-checking there covers a related but distinct
cross-site-WebSocket-hijacking concern).

## What still needs a live environment
- Actual SQLi fuzzing against a running Postgres instance
- A real browser session to confirm no XSS vector was missed by static
  review
- Load-testing the rate limiter under real concurrent traffic
