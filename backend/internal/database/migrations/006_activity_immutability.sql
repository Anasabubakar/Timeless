-- Audit log immutability: activities can be created and hard-deleted
-- (by the retention job), but never modified in place once written.
-- (Applied at boot via database.enforceActivityImmutability in
-- internal/database/migrate.go; this file documents the same change.)

CREATE OR REPLACE FUNCTION reject_activity_update() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'activities are immutable: row % cannot be updated after creation', OLD.id;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS activities_immutable ON activities;
CREATE TRIGGER activities_immutable
    BEFORE UPDATE ON activities
    FOR EACH ROW
    EXECUTE FUNCTION reject_activity_update();
