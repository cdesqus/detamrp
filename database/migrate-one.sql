\set ON_ERROR_STOP on
BEGIN;
SELECT pg_advisory_xact_lock(hashtext('order-stock-schema-migrations'));
SELECT NOT EXISTS (SELECT 1 FROM schema_migrations WHERE version = :migration_version) AS apply_migration \gset
\if :apply_migration
\i :migration_file
INSERT INTO schema_migrations(version, name) VALUES (:migration_version, :'migration_name');
\endif
COMMIT;
