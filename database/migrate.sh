#!/bin/sh
set -eu

: "${PGHOST:?PGHOST is required}"
: "${PGDATABASE:?PGDATABASE is required}"
: "${PGUSER:?PGUSER is required}"
: "${PGPASSWORD:?PGPASSWORD is required}"

legacy_initialized="$(psql -X -A -t -v ON_ERROR_STOP=1 -c "SELECT to_regclass('public.tenants') IS NOT NULL")"

psql -X -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
  version integer PRIMARY KEY,
  name text NOT NULL,
  applied_at timestamptz NOT NULL DEFAULT now()
);
SQL

recorded_migrations="$(psql -X -A -t -v ON_ERROR_STOP=1 -c 'SELECT count(*) FROM schema_migrations')"
if [ "$legacy_initialized" = "t" ] && [ "$recorded_migrations" = "0" ]; then
  psql -X -v ON_ERROR_STOP=1 <<'SQL'
INSERT INTO schema_migrations(version, name) VALUES
  (1, '001_foundation.sql'),
  (2, '002_master_data.sql'),
  (3, '003_raw_material_price.sql'),
  (4, '004_settings_identity.sql')
ON CONFLICT (version) DO NOTHING;
SQL
fi

find /migrations -maxdepth 1 -type f -name '[0-9][0-9][0-9]_*.sql' | sort -n | while IFS= read -r migration_file; do
  migration_name="$(basename "$migration_file")"
  migration_prefix="${migration_name%%_*}"
  migration_version="$(printf '%s' "$migration_prefix" | sed 's/^0*//')"
  psql -X -v ON_ERROR_STOP=1 \
    -v migration_version="$migration_version" \
    -v migration_name="$migration_name" \
    -v migration_file="$migration_file" \
    -f /migration-tools/migrate-one.sql
done
