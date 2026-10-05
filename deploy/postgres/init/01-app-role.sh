#!/bin/sh
# Runs once, when the Postgres data volume is first created. Creates the
# application role: it can log in but can't own objects or bypass row-level
# security. Migrations run as the owner role (POSTGRES_USER).
set -eu

psql -v ON_ERROR_STOP=1 \
  --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  -v app_password="$SOMA_APP_DB_PASSWORD" <<'EOSQL'
CREATE ROLE soma_app LOGIN PASSWORD :'app_password'
  NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
EOSQL
