-- name: UpsertUser :exec
-- users has no row-level security, so an upsert is safe here.
INSERT INTO users (id, display_name, created_at)
VALUES (@id, @display_name, @now)
ON CONFLICT (id) DO NOTHING;

-- name: InsertSpace :exec
-- Plain INSERT on purpose: ON CONFLICT would also apply the space's read
-- policy to the new row, which fails before the owner is a member (ADR-014).
INSERT INTO spaces (id, name, created_by, created_at)
VALUES (@id, @name, @created_by, @now);

-- name: InsertSpaceMember :exec
INSERT INTO space_members (space_id, user_id, role, created_at)
VALUES (@space_id, @user_id, @role, @now);
