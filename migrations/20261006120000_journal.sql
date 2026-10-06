-- Journal entries (JRN-01..04). Private: an entry is visible only to its author,
-- even within a shared space (business spec, journal rule). Enforced by RLS.

-- +goose Up
CREATE TABLE journal_entries (
    id         uuid PRIMARY KEY REFERENCES entities (id) ON DELETE CASCADE,
    space_id   uuid NOT NULL REFERENCES spaces (id),
    author_id  uuid NOT NULL REFERENCES users (id),
    entry_date date NOT NULL,
    title      text CHECK (char_length(title) <= 200),
    body       text NOT NULL DEFAULT '' CHECK (char_length(body) <= 100000),
    mood       smallint CHECK (mood BETWEEN 1 AND 5)
);
CREATE INDEX journal_entries_space_idx ON journal_entries (space_id);
CREATE INDEX journal_entries_author_idx ON journal_entries (author_id, entry_date DESC);

ALTER TABLE journal_entries ENABLE ROW LEVEL SECURITY;
CREATE POLICY journal_entries_read ON journal_entries FOR SELECT
    USING (author_id = soma_current_user_id()
           AND space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id()));
CREATE POLICY journal_entries_insert ON journal_entries FOR INSERT
    WITH CHECK (author_id = soma_current_user_id()
                AND space_id IN (SELECT space_id FROM space_members
                                 WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY journal_entries_update ON journal_entries FOR UPDATE
    USING (author_id = soma_current_user_id())
    WITH CHECK (author_id = soma_current_user_id());
CREATE POLICY journal_entries_delete ON journal_entries FOR DELETE
    USING (author_id = soma_current_user_id());

-- +goose Down
DROP TABLE journal_entries;
