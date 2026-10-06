-- The Self profile (SELF-01, SELF-03, SELF-04): one per space, created with the
-- space. Personality tests and life story come later (SELF-02, SELF-05).

-- +goose Up
CREATE TABLE self_profiles (
    space_id       uuid PRIMARY KEY REFERENCES spaces (id),
    preferred_name text CHECK (char_length(preferred_name) <= 100),
    birth_date     date,
    core_values    text CHECK (char_length(core_values) <= 2000),
    bio            text CHECK (char_length(bio) <= 5000),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    version        integer NOT NULL DEFAULT 1 CHECK (version >= 1)
);

ALTER TABLE self_profiles ENABLE ROW LEVEL SECURITY;
CREATE POLICY self_profiles_read ON self_profiles FOR SELECT
    USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id()));
CREATE POLICY self_profiles_insert ON self_profiles FOR INSERT
    WITH CHECK (space_id IN (SELECT space_id FROM space_members
                             WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY self_profiles_update ON self_profiles FOR UPDATE
    USING (space_id IN (SELECT space_id FROM space_members
                        WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')))
    WITH CHECK (space_id IN (SELECT space_id FROM space_members
                             WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));

-- +goose Down
DROP TABLE self_profiles;
