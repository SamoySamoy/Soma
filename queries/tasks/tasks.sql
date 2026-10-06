-- name: InsertTask :exec
INSERT INTO tasks (id, space_id, title, notes, due_on)
VALUES (@id, @space_id, @title, @notes, @due_on);

-- name: GetTask :one
SELECT t.id, t.title, t.notes, t.due_on, t.completed_at, e.created_at, e.updated_at, e.version
FROM tasks t
JOIN entities e ON e.id = t.id
WHERE t.id = @id AND t.space_id = @space_id AND e.deleted_at IS NULL;

-- name: ListTasks :many
-- Newest saved first, so the cursor (the last ID seen) keeps paging stable.
-- The browser sorts open tasks by due date.
SELECT t.id, t.title, t.notes, t.due_on, t.completed_at, e.created_at, e.updated_at, e.version
FROM tasks t
JOIN entities e ON e.id = t.id
WHERE t.space_id = @space_id AND e.deleted_at IS NULL
  AND (sqlc.narg('before')::uuid IS NULL OR t.id < sqlc.narg('before')::uuid)
  AND (sqlc.arg('only_open')::boolean = false OR t.completed_at IS NULL)
ORDER BY t.id DESC
LIMIT @row_limit;

-- name: UpdateTask :exec
UPDATE tasks
SET title = @title, notes = @notes, due_on = @due_on
WHERE id = @id AND space_id = @space_id;

-- name: SetTaskCompleted :exec
UPDATE tasks
SET completed_at = @completed_at
WHERE id = @id AND space_id = @space_id;

-- name: TaskCounts :one
-- Open tasks overdue (due before today) and due today, for the Responsibilities area.
SELECT
  count(*) FILTER (WHERE t.completed_at IS NULL AND t.due_on < sqlc.arg('today')::date)::bigint AS overdue,
  count(*) FILTER (WHERE t.completed_at IS NULL AND t.due_on = sqlc.arg('today')::date)::bigint AS due_today,
  count(*) FILTER (WHERE t.completed_at IS NULL AND t.due_on < sqlc.arg('today')::date - 3)::bigint AS overdue_long
FROM tasks t
JOIN entities e ON e.id = t.id
WHERE t.space_id = @space_id AND e.deleted_at IS NULL;
