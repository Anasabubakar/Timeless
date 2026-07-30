# Risk Assessment

Residual risk after this hardening pass, rated Likelihood x Impact.
"Residual" — these are what's left, not a list of everything fixed
(see `SECURITY_REPORT.md` for the fixed-items log).

| # | Risk | Likelihood | Impact | Residual severity | Mitigation status |
|---|------|-----------|--------|--------------------|--------------------|
| 1 | No live pentest run | High (certain — none has run) | Unknown until done | **Medium** | Everything below is code-level review; a real dynamic test could surface something this process couldn't |
| 2 | Fiber pinned to a beta release | Low (beta.4 is stable in practice, just unofficial) | Medium (missing whatever hardening went into the 1.0 cut) | **Low-Medium** | Documented in DEPENDENCY_AUDIT.md; needs a live env to regression-test the upgrade, not done blind |
| 3 | No bot-detection/CAPTCHA on registration/login | Medium (credential-stuffing bots are common against any public signup) | Medium (rate limiting + lockout already blunt this significantly) | **Low-Medium** | Rate limiting + account lockout in place; CAPTCHA is an infra/product decision, not added speculatively |
| 4 | No WAF/edge DDoS layer | Medium (depends entirely on deployment target) | Medium-High for a large-scale flood | **Low** (deployment-dependent) | Application-level rate limiting exists; edge protection is the hosting platform's job (Cloudflare/Vercel/etc.) |
| 5 | Read-only data access isn't audited | Medium (a valid session reading more than it should is a real insider-threat shape) | Medium (data exposure, not data corruption) | **Low-Medium** | Rate limiting bounds volume; full read-audit would add significant storage/noise — a deliberate scope decision, not an oversight |
| 6 | `lib/pq` in maintenance mode | Low (stable, just not actively developed) | Low (used only for a type helper, not the connection) | **Low** | Documented; migration path identified, deferred (needs live DB to verify array serialization doesn't regress) |
| 7 | DELETE still possible on the immutable audit table | Low (needs direct DB access, which is itself a high-bar compromise) | Medium (could hide evidence of an intrusion) | **Low** | UPDATE is trigger-blocked; DELETE is needed for the opt-in retention purge and is bounded by DB-credential-level access control |
| 8 | Live dependency-vulnerability scan not run this session | Medium (sandbox network was unreliable for this specific check) | Depends entirely on what a scan would find | **Low-Medium** | Manual review found nothing outdated/vulnerable at reviewed versions; `govulncheck`/`npm audit` need to run in CI going forward |

## Accepted risks

Risks 4 and 6 are accepted as-is for this pass: #4 is inherently a
deployment-platform decision, not something this codebase can fully
own; #6 has a clear migration path but touches live data
serialization in a way that genuinely needs a real database to verify
safely, which this sandbox didn't have.

## Requires action before a production launch

- **#1**: commission an actual penetration test or dynamic security
  scan against a staged deployment.
- **#3/#4**: decide on and provision bot-detection/WAF at the chosen
  hosting platform.
- **#8**: wire `govulncheck ./...` and `npm audit --omit=dev` into CI.

## Everything else

Tracked as backlog, not launch-blocking, per the severity above.
