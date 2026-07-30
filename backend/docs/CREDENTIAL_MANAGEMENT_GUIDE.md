# Credential Management Guide

## What Timeless holds
- `JWT_SECRET` (+ `JWT_SECRET_PREVIOUS`): signs/verifies all access
  and refresh tokens.
- `CREDENTIALS_ENCRYPTION_KEY` (+ `..._PREVIOUS`): encrypts OAuth
  tokens and integration credentials at rest. Must be distinct from
  `JWT_SECRET` in production — `Config.Validate()` enforces this.
- Third-party OAuth app secrets: `NOTION_CLIENT_SECRET`,
  `APOLLO_CLIENT_SECRET`.
- `DATABASE_URL`, `REDIS_URL` (may embed credentials).
- SMTP/SendGrid credentials for outbound email.
- S3/MinIO access keys for file storage.
- Per-org, per-user secrets: user passwords (bcrypt hashes only, never
  plaintext), MFA secrets (encrypted with `CredentialsEncryptionKey`),
  OAuth tokens per connected integration (same).

## Storage rules
- Never in source control — `.env` is gitignored; only `.env.example`
  (with placeholder values) is tracked. This pass grepped for and
  removed dead config (`GOOGLE_CLIENT_*`) rather than leave unused
  secret slots around.
- Application secrets (`JWT_SECRET`, `CREDENTIALS_ENCRYPTION_KEY`, OAuth
  app secrets, DB/Redis URLs) belong in the deployment platform's
  secret manager, injected as env vars — not baked into an image.
- User/integration secrets never leave the database except: (a)
  decrypted transiently in memory to make an API call on the user's
  behalf, (b) a backup code or MFA secret shown once at enrollment
  time to the user themselves, never persisted in plaintext.

## Rotation
- **JWT_SECRET**: set a new value, move the old one to the front of
  `JWT_SECRET_PREVIOUS`. Outstanding tokens signed under the old key
  keep verifying (via `JWTKeyring`) until they naturally expire; new
  tokens sign under the new key. Drop the old value from
  `_PREVIOUS` only once you're confident nothing still holds a token
  signed under it (>= the longest refresh token lifetime,
  `REMEMBER_ME_EXPIRY` if used).
- **CREDENTIALS_ENCRYPTION_KEY**: same pattern, then call
  `POST /integrations/rotate-credentials` to re-encrypt every stored
  credential under the new key — until that runs, old rows stay
  encrypted under the retired key (still decryptable via `_PREVIOUS`,
  just not yet migrated).
- **OAuth app secrets** (Notion/Apollo): rotate in the provider's
  developer console, update the env var. Existing connected
  integrations keep working (the *user's* token, not the app secret,
  is what's stored per-integration) until the provider explicitly
  requires re-consent.
- **Passwords**: user-initiated via `/profile/password` or
  `/auth/reset-password` — both revoke every other session for that
  user as part of the change, not just update the hash.

## Never log
`internal/logging`'s redacting writer strips `Authorization` headers,
`password`/`secret`/`token`/`api_key`-shaped key-value pairs, and
client-secret patterns before anything reaches stdout — verified with
tests (`internal/logging/redact_test.go`). Still, treat this as
defense-in-depth, not a license to log raw request bodies "because
it's redacted anyway" — don't log secrets in the first place where
avoidable.

## Emergency revocation
See `INCIDENT_RESPONSE_PLAN.md`'s Containment section.
