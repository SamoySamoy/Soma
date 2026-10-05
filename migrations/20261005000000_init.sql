-- Baseline: extensions and privileges for the application role.
-- The soma_app role is created outside migrations (deploy/postgres/init and
-- the test helper) because roles are cluster-wide and need superuser rights.

-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;

GRANT USAGE ON SCHEMA public TO soma_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO soma_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO soma_app;

-- Lets /readyz compare the applied schema version with the binary's.
GRANT SELECT ON goose_db_version TO soma_app;

-- +goose Down
REVOKE SELECT ON goose_db_version FROM soma_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE USAGE, SELECT ON SEQUENCES FROM soma_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM soma_app;
REVOKE USAGE ON SCHEMA public FROM soma_app;
DROP EXTENSION IF EXISTS unaccent;
DROP EXTENSION IF EXISTS pg_trgm;
