-- name: InsertEntry :exec
INSERT INTO journal_entries (id, space_id, author_id, entry_date, title, body, mood)
VALUES (@id, @space_id, @author_id, @entry_date, @title, @body, @mood);

-- name: GetEntry :one
SELECT j.id, j.entry_date, j.title, j.body, j.mood, e.created_at, e.updated_at, e.version
FROM journal_entries j
JOIN entities e ON e.id = j.id
WHERE j.id = @id AND j.space_id = @space_id AND j.author_id = @author_id AND e.deleted_at IS NULL;

-- name: ListEntries :many
-- Newest saved first. UUIDv7 keys are time-ordered, so the cursor is the last ID seen.
SELECT j.id, j.entry_date, j.title, j.body, j.mood, e.created_at, e.updated_at, e.version
FROM journal_entries j
JOIN entities e ON e.id = j.id
WHERE j.space_id = @space_id AND j.author_id = @author_id AND e.deleted_at IS NULL
  AND (sqlc.narg('before')::uuid IS NULL OR j.id < sqlc.narg('before')::uuid)
ORDER BY j.id DESC
LIMIT @row_limit;

-- name: UpdateEntry :exec
UPDATE journal_entries
SET entry_date = @entry_date, title = @title, body = @body, mood = @mood
WHERE id = @id AND space_id = @space_id AND author_id = @author_id;

-- name: LastEntryStats :one
-- The newest entry date and how many entries exist, for the Mind area.
SELECT max(j.entry_date)::date AS last_entry_date, count(*)::bigint AS entries
FROM journal_entries j
JOIN entities e ON e.id = j.id
WHERE j.space_id = @space_id AND j.author_id = @author_id AND e.deleted_at IS NULL;
