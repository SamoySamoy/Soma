-- Spaces, membership, the entity spine and the first record type: people
-- contacts (PPL-01). Every record table has space_id and row-level security.
-- See ADR-002 (entity spine), ADR-008 (two-layer authorization) and ADR-014.

-- +goose Up

-- The acting user for the current transaction, or NULL when none is set.
-- Policies call this; db.InTx sets soma.user_id on every transaction.
CREATE FUNCTION soma_current_user_id() RETURNS uuid
    LANGUAGE sql STABLE
    AS $$ SELECT NULLIF(current_setting('soma.user_id', true), '')::uuid $$;

CREATE TABLE users (
    id           uuid PRIMARY KEY DEFAULT uuidv7(),
    display_name text NOT NULL CHECK (char_length(display_name) BETWEEN 1 AND 100),
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE spaces (
    id         uuid PRIMARY KEY DEFAULT uuidv7(),
    name       text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    created_by uuid NOT NULL REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX spaces_created_by_idx ON spaces (created_by);

CREATE TABLE space_members (
    space_id   uuid NOT NULL REFERENCES spaces (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id),
    role       text NOT NULL CHECK (role IN ('owner', 'editor', 'viewer')),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (space_id, user_id)
);
CREATE INDEX space_members_user_idx ON space_members (user_id);

CREATE TABLE entities (
    id         uuid PRIMARY KEY DEFAULT uuidv7(),
    space_id   uuid NOT NULL REFERENCES spaces (id),
    kind       text NOT NULL CHECK (kind ~ '^[a-z]+\.[a-z_]+$'),
    created_by uuid NOT NULL REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    version    integer NOT NULL DEFAULT 1 CHECK (version >= 1)
);
CREATE INDEX entities_space_kind_idx ON entities (space_id, kind, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX entities_created_by_idx ON entities (created_by);

CREATE TABLE people_contacts (
    id           uuid PRIMARY KEY REFERENCES entities (id) ON DELETE CASCADE,
    space_id     uuid NOT NULL REFERENCES spaces (id),
    display_name text NOT NULL CHECK (char_length(display_name) BETWEEN 1 AND 200),
    nickname     text CHECK (char_length(nickname) <= 100),
    email        text CHECK (char_length(email) <= 254),
    phone        text CHECK (char_length(phone) <= 40),
    birthday     date,
    how_we_met   text CHECK (char_length(how_we_met) <= 500),
    notes        text CHECK (char_length(notes) <= 10000)
);
CREATE INDEX people_contacts_space_idx ON people_contacts (space_id);

-- Row-level security. Reads need membership; writes need an owner or editor
-- role. Viewers therefore fail at the database even if a service forgets
-- its own check.
ALTER TABLE spaces ENABLE ROW LEVEL SECURITY;
CREATE POLICY spaces_read ON spaces FOR SELECT
    USING (id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id()));
CREATE POLICY spaces_create ON spaces FOR INSERT
    WITH CHECK (created_by = soma_current_user_id());

ALTER TABLE space_members ENABLE ROW LEVEL SECURITY;
CREATE POLICY space_members_own ON space_members FOR ALL
    USING (user_id = soma_current_user_id())
    WITH CHECK (user_id = soma_current_user_id());

ALTER TABLE entities ENABLE ROW LEVEL SECURITY;
CREATE POLICY entities_read ON entities FOR SELECT
    USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id()));
-- Inserts must name the acting user as creator. Updates and deletes may touch
-- any record in the space: who may edit a given record is the service's call.
CREATE POLICY entities_insert ON entities FOR INSERT
    WITH CHECK (space_id IN (SELECT space_id FROM space_members
                             WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor'))
                AND created_by = soma_current_user_id());
CREATE POLICY entities_update ON entities FOR UPDATE
    USING (space_id IN (SELECT space_id FROM space_members
                        WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')))
    WITH CHECK (space_id IN (SELECT space_id FROM space_members
                             WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY entities_delete ON entities FOR DELETE
    USING (space_id IN (SELECT space_id FROM space_members
                        WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));

ALTER TABLE people_contacts ENABLE ROW LEVEL SECURITY;
CREATE POLICY people_contacts_read ON people_contacts FOR SELECT
    USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id()));
CREATE POLICY people_contacts_insert ON people_contacts FOR INSERT
    WITH CHECK (space_id IN (SELECT space_id FROM space_members
                             WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY people_contacts_update ON people_contacts FOR UPDATE
    USING (space_id IN (SELECT space_id FROM space_members
                        WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')))
    WITH CHECK (space_id IN (SELECT space_id FROM space_members
                             WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY people_contacts_delete ON people_contacts FOR DELETE
    USING (space_id IN (SELECT space_id FROM space_members
                        WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));

-- +goose Down
DROP TABLE people_contacts;
DROP TABLE entities;
-- spaces' read policy depends on space_members, so the policies go first.
DROP POLICY spaces_read ON spaces;
DROP POLICY spaces_create ON spaces;
DROP TABLE space_members;
DROP TABLE spaces;
DROP TABLE users;
DROP FUNCTION soma_current_user_id();
