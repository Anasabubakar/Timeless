-- users.email had no index at all despite being the lookup key for
-- every single login attempt (UserRepository.FindByEmail) — a full
-- table scan on every login. Not unique: Register's "already
-- registered" check is global but InviteMember's uniqueness check is
-- scoped per-org, so existing mixed semantics are preserved rather than
-- redesigned as part of an indexing fix.
-- (Applied at boot via GORM AutoMigrate; this file documents the same change.)

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
