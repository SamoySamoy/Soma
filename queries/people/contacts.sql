-- name: InsertContact :exec
INSERT INTO people_contacts (id, space_id, display_name, nickname, email, phone, birthday, how_we_met, notes)
VALUES (@id, @space_id, @display_name, @nickname, @email, @phone, @birthday, @how_we_met, @notes);

-- name: GetContact :one
SELECT c.id, c.display_name, c.nickname, c.email, c.phone, c.birthday, c.how_we_met, c.notes,
       e.created_at, e.updated_at, e.version
FROM people_contacts c
JOIN entities e ON e.id = c.id
WHERE c.id = @id AND c.space_id = @space_id AND e.deleted_at IS NULL;

-- name: ListContacts :many
-- Newest first. UUIDv7 keys are time-ordered, so the cursor is the last ID seen.
SELECT c.id, c.display_name, c.nickname, c.email, c.phone, c.birthday, c.how_we_met, c.notes,
       e.created_at, e.updated_at, e.version
FROM people_contacts c
JOIN entities e ON e.id = c.id
WHERE c.space_id = @space_id
  AND e.deleted_at IS NULL
  AND (sqlc.narg('before')::uuid IS NULL OR c.id < sqlc.narg('before')::uuid)
ORDER BY c.id DESC
LIMIT @row_limit;

-- name: UpdateContact :exec
UPDATE people_contacts
SET display_name = @display_name,
    nickname = @nickname,
    email = @email,
    phone = @phone,
    birthday = @birthday,
    how_we_met = @how_we_met,
    notes = @notes
WHERE id = @id AND space_id = @space_id;

-- name: ListBirthdays :many
SELECT c.birthday
FROM people_contacts c
JOIN entities e ON e.id = c.id
WHERE c.space_id = @space_id
  AND e.deleted_at IS NULL
  AND c.birthday IS NOT NULL;
