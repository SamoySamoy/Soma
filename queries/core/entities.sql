-- name: InsertEntity :one
INSERT INTO entities (space_id, kind, created_by, created_at, updated_at)
VALUES (@space_id, @kind, @created_by, @now, @now)
RETURNING id, version;

-- name: BumpEntityVersion :one
-- Returns no rows when the expected version is stale, which the caller maps to 412.
UPDATE entities
SET version = version + 1, updated_at = @now
WHERE id = @id AND version = @expected_version AND deleted_at IS NULL
RETURNING version;

-- name: SoftDeleteEntity :execrows
UPDATE entities
SET deleted_at = @now, updated_at = @now, version = version + 1
WHERE id = @id AND deleted_at IS NULL;
