-- Tasks (TSK-01..02). Recurrence and reminders come later (TSK-03..04).

-- +goose Up
CREATE TABLE tasks (
    id           uuid PRIMARY KEY REFERENCES entities (id) ON DELETE CASCADE,
    space_id     uuid NOT NULL REFERENCES spaces (id),
    title        text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
    notes        text CHECK (char_length(notes) <= 5000),
    due_on       date,
    completed_at timestamptz
);
CREATE INDEX tasks_space_due_idx ON tasks (space_id, due_on);

ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
CREATE POLICY tasks_read ON tasks FOR SELECT
    USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id()));
CREATE POLICY tasks_insert ON tasks FOR INSERT
    WITH CHECK (space_id IN (SELECT space_id FROM space_members
                             WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY tasks_update ON tasks FOR UPDATE
    USING (space_id IN (SELECT space_id FROM space_members
                        WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')))
    WITH CHECK (space_id IN (SELECT space_id FROM space_members
                             WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY tasks_delete ON tasks FOR DELETE
    USING (space_id IN (SELECT space_id FROM space_members
                        WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));

-- +goose Down
DROP TABLE tasks;
