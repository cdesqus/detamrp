BEGIN;

CREATE TABLE IF NOT EXISTS schema_migrations (
  version integer PRIMARY KEY,
  name text NOT NULL,
  applied_at timestamptz NOT NULL DEFAULT now()
);

DO $$
DECLARE
  legacy_present boolean;
  recorded_migrations integer;
  version_001_complete boolean := false;
  version_002_complete boolean := false;
  version_003_complete boolean := false;
  version_004_complete boolean := false;
  identity_tables_complete boolean;
BEGIN
  legacy_present := to_regclass(current_schema() || '.tenants') IS NOT NULL
    OR to_regclass(current_schema() || '.tenant_settings') IS NOT NULL
    OR to_regclass(current_schema() || '.users') IS NOT NULL
    OR to_regclass(current_schema() || '.sessions') IS NOT NULL
    OR to_regclass(current_schema() || '.permissions') IS NOT NULL
    OR to_regclass(current_schema() || '.roles') IS NOT NULL
    OR to_regclass(current_schema() || '.user_roles') IS NOT NULL
    OR to_regclass(current_schema() || '.role_permissions') IS NOT NULL
    OR to_regclass(current_schema() || '.measurements') IS NOT NULL
    OR to_regclass(current_schema() || '.suppliers') IS NOT NULL
    OR to_regclass(current_schema() || '.raw_materials') IS NOT NULL
    OR to_regclass(current_schema() || '.warehouses') IS NOT NULL
    OR to_regclass(current_schema() || '.warehouse_locations') IS NOT NULL;

  SELECT count(*) INTO recorded_migrations FROM schema_migrations;
  IF NOT legacy_present OR recorded_migrations > 0 THEN
    RETURN;
  END IF;

  identity_tables_complete :=
    to_regclass(current_schema() || '.tenants') IS NOT NULL AND
    to_regclass(current_schema() || '.tenant_settings') IS NOT NULL AND
    to_regclass(current_schema() || '.users') IS NOT NULL AND
    to_regclass(current_schema() || '.sessions') IS NOT NULL AND
    to_regclass(current_schema() || '.permissions') IS NOT NULL AND
    to_regclass(current_schema() || '.roles') IS NOT NULL AND
    to_regclass(current_schema() || '.user_roles') IS NOT NULL AND
    to_regclass(current_schema() || '.role_permissions') IS NOT NULL;
  IF identity_tables_complete THEN
    SELECT EXISTS (SELECT 1 FROM permissions WHERE code = 'po.view')
       AND EXISTS (SELECT 1 FROM permissions WHERE code = 'po.price.view')
      INTO version_001_complete;
  END IF;

  version_002_complete :=
    to_regclass(current_schema() || '.measurements') IS NOT NULL AND
    to_regclass(current_schema() || '.suppliers') IS NOT NULL AND
    to_regclass(current_schema() || '.raw_materials') IS NOT NULL AND
    to_regclass(current_schema() || '.warehouses') IS NOT NULL AND
    to_regclass(current_schema() || '.warehouse_locations') IS NOT NULL;

  SELECT count(*) = 2 INTO version_003_complete
  FROM information_schema.columns
  WHERE table_schema = current_schema()
    AND table_name = 'raw_materials'
    AND column_name IN ('standard_unit_price', 'currency');

  SELECT count(*) = 6 INTO version_004_complete
  FROM information_schema.columns
  WHERE table_schema = current_schema()
    AND ((table_name = 'users' AND column_name = 'email')
      OR (table_name = 'roles' AND column_name IN ('active', 'created_by_user_id', 'updated_by_user_id', 'updated_at'))
      OR (table_name = 'tenant_settings' AND column_name = 'default_approver_user_id'));
  version_004_complete := version_004_complete
    AND to_regclass(current_schema() || '.users_tenant_email_ci_key') IS NOT NULL;

  IF NOT (version_001_complete AND version_002_complete AND version_003_complete AND version_004_complete) THEN
    RAISE EXCEPTION 'partial legacy schema: refusing baseline (001=%, 002=%, 003=%, 004=%)',
      version_001_complete, version_002_complete, version_003_complete, version_004_complete;
  END IF;

  INSERT INTO schema_migrations(version, name) VALUES
    (1, '001_foundation.sql'),
    (2, '002_master_data.sql'),
    (3, '003_raw_material_price.sql'),
    (4, '004_settings_identity.sql')
  ON CONFLICT (version) DO NOTHING;
END
$$;

COMMIT;
