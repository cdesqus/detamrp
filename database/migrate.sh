#!/bin/sh
set -eu

: "${PGHOST:?PGHOST is required}"
: "${PGDATABASE:?PGDATABASE is required}"
: "${PGUSER:?PGUSER is required}"
: "${PGPASSWORD:?PGPASSWORD is required}"

psql -X -v ON_ERROR_STOP=1 -f /migration-tools/migration-bootstrap.sql

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
