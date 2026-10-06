-- name: GetSelfProfile :one
SELECT preferred_name, birth_date, core_values, bio, updated_at, version
FROM self_profiles
WHERE space_id = @space_id;

-- name: UpdateSelfProfile :one
-- Returns no rows when the expected version is stale, which the caller maps to 412.
UPDATE self_profiles
SET preferred_name = @preferred_name,
    birth_date = @birth_date,
    core_values = @core_values,
    bio = @bio,
    updated_at = @now,
    version = version + 1
WHERE space_id = @space_id AND version = @expected_version
RETURNING version;
