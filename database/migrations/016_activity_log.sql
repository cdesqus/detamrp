BEGIN;

CREATE TABLE activity_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  occurred_at timestamptz NOT NULL DEFAULT now(),
  actor_user_id uuid,
  actor_name text NOT NULL DEFAULT 'System',
  module text NOT NULL,
  action text NOT NULL,
  target_type text NOT NULL,
  target_id uuid,
  target_code text NOT NULL DEFAULT '',
  before_data jsonb,
  after_data jsonb,
  UNIQUE (tenant_id, id)
);

CREATE INDEX activity_logs_tenant_occurred_idx
  ON activity_logs (tenant_id, occurred_at DESC, id DESC);
CREATE INDEX activity_logs_tenant_actor_idx
  ON activity_logs (tenant_id, actor_user_id, occurred_at DESC);
CREATE INDEX activity_logs_tenant_module_action_idx
  ON activity_logs (tenant_id, module, action, occurred_at DESC);

ALTER TABLE activity_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE activity_logs FORCE ROW LEVEL SECURITY;
CREATE POLICY activity_logs_isolation ON activity_logs
  FOR SELECT
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
CREATE POLICY activity_logs_trigger_insert ON activity_logs
  FOR INSERT
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE OR REPLACE FUNCTION reject_activity_log_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION 'activity log is append-only' USING ERRCODE = '23514';
END;
$$;

CREATE TRIGGER activity_logs_append_only
BEFORE UPDATE OR DELETE ON activity_logs
FOR EACH ROW EXECUTE FUNCTION reject_activity_log_mutation();

CREATE OR REPLACE FUNCTION sanitize_activity_snapshot(snapshot jsonb)
RETURNS jsonb
LANGUAGE sql
IMMUTABLE
AS $$
  SELECT CASE
    WHEN snapshot IS NULL THEN NULL
    ELSE snapshot - ARRAY[
      'password_hash',
      'smtp_password_encrypted',
      'company_logo',
      'company_logo_mime',
      'login_background',
      'login_background_mime',
      'token_hash',
      'payload'
    ]::text[]
  END;
$$;

CREATE OR REPLACE FUNCTION record_activity_change()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
  old_row jsonb := CASE WHEN TG_OP = 'INSERT' THEN NULL ELSE to_jsonb(OLD) END;
  new_row jsonb := CASE WHEN TG_OP = 'DELETE' THEN NULL ELSE to_jsonb(NEW) END;
  source_row jsonb := COALESCE(new_row, old_row);
  activity_tenant_id uuid;
  activity_actor_id uuid;
  activity_actor_name text := 'System';
  activity_module text;
  activity_action text;
  activity_target_code text;
  actor_setting text;
  old_status text;
  new_status text;
BEGIN
  activity_tenant_id := (source_row->>'tenant_id')::uuid;
  actor_setting := NULLIF(current_setting('app.user_id', true), '');

  IF actor_setting IS NOT NULL
     AND actor_setting ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
     AND actor_setting <> '00000000-0000-0000-0000-000000000000' THEN
    activity_actor_id := actor_setting::uuid;
  ELSE
    activity_actor_id := NULLIF(
      COALESCE(
        source_row->>'updated_by_user_id',
        source_row->>'completed_by_user_id',
        source_row->>'created_by_user_id'
      ),
      ''
    )::uuid;
  END IF;

  IF activity_actor_id IS NOT NULL THEN
    SELECT COALESCE(NULLIF(BTRIM(display_name), ''), username)
      INTO activity_actor_name
      FROM users
     WHERE tenant_id = activity_tenant_id AND id = activity_actor_id;
    activity_actor_name := COALESCE(activity_actor_name, 'System');
  END IF;

  activity_module := CASE TG_TABLE_NAME
    WHEN 'units' THEN 'DATA_MASTER'
    WHEN 'categories' THEN 'DATA_MASTER'
    WHEN 'packings' THEN 'DATA_MASTER'
    WHEN 'plants' THEN 'DATA_MASTER'
    WHEN 'suppliers' THEN 'DATA_MASTER'
    WHEN 'raw_materials' THEN 'DATA_MASTER'
    WHEN 'purchase_orders' THEN 'PROCUREMENT'
    WHEN 'delivery_notes' THEN 'LOGISTICS'
    WHEN 'receiving_sessions' THEN 'RECEIVING'
    WHEN 'receivings' THEN 'RECEIVING'
    WHEN 'outgoing_sessions' THEN 'OUTGOING'
    WHEN 'outgoing_documents' THEN 'OUTGOING'
    WHEN 'inventory_ledger_entries' THEN 'INVENTORY'
    WHEN 'kanban_lots' THEN 'INVENTORY'
    ELSE 'SETTINGS'
  END;

  activity_target_code := COALESCE(
    source_row->>'po_number',
    source_row->>'delivery_note_number',
    source_row->>'receiving_number',
    source_row->>'document_number',
    source_row->>'kanban_id',
    source_row->>'permission_code',
    source_row->>'code',
    source_row->>'username',
    source_row->>'company_name',
    source_row->>'id',
    ''
  );

  old_status := old_row->>'status';
  new_status := new_row->>'status';

  IF TG_TABLE_NAME = 'tenant_settings'
     AND old_row->'company_logo' IS DISTINCT FROM new_row->'company_logo' THEN
    activity_action := 'COMPANY_LOGO_UPDATED';
  ELSIF TG_TABLE_NAME = 'tenant_settings'
     AND old_row->'login_background' IS DISTINCT FROM new_row->'login_background' THEN
    activity_action := 'LOGIN_BACKGROUND_UPDATED';
  ELSIF TG_OP = 'DELETE' THEN
    activity_action := 'DELETED';
  ELSIF TG_OP = 'INSERT' AND TG_TABLE_NAME = 'delivery_notes' THEN
    activity_action := 'ISSUED';
  ELSIF TG_OP = 'INSERT' AND TG_TABLE_NAME IN ('receivings', 'outgoing_documents') THEN
    activity_action := 'COMPLETED';
  ELSIF TG_OP = 'INSERT' AND TG_TABLE_NAME = 'inventory_ledger_entries' THEN
    activity_action := CASE new_row->>'event_type'
      WHEN 'RECEIVING' THEN 'RECEIVED'
      ELSE 'MOVED'
    END;
  ELSIF TG_OP = 'INSERT' THEN
    activity_action := 'CREATED';
  ELSIF old_row ? 'active'
     AND (old_row->>'active')::boolean IS DISTINCT FROM (new_row->>'active')::boolean THEN
    activity_action := CASE WHEN (new_row->>'active')::boolean THEN 'ACTIVATED' ELSE 'DEACTIVATED' END;
  ELSIF old_status IS DISTINCT FROM new_status THEN
    activity_action := CASE new_status
      WHEN 'PENDING_APPROVAL' THEN 'SUBMITTED'
      WHEN 'APPROVED' THEN 'APPROVED'
      WHEN 'REJECTED' THEN 'REJECTED'
      WHEN 'CANCELLED' THEN 'CANCELLED'
      WHEN 'COMPLETED' THEN 'COMPLETED'
      WHEN 'IN_STOCK' THEN 'RECEIVED'
      WHEN 'CONSUMED' THEN 'MOVED'
      WHEN 'PARTIALLY_RECEIVED' THEN 'RECEIVED'
      WHEN 'FULLY_RECEIVED' THEN 'RECEIVED'
      ELSE 'UPDATED'
    END;
  ELSE
    activity_action := 'UPDATED';
  END IF;

  INSERT INTO activity_logs (
    tenant_id,
    actor_user_id,
    actor_name,
    module,
    action,
    target_type,
    target_id,
    target_code,
    before_data,
    after_data
  ) VALUES (
    activity_tenant_id,
    activity_actor_id,
    activity_actor_name,
    activity_module,
    activity_action,
    TG_TABLE_NAME,
    NULLIF(COALESCE(source_row->>'id', source_row->>'role_id', source_row->>'user_id'), '')::uuid,
    activity_target_code,
    sanitize_activity_snapshot(old_row),
    sanitize_activity_snapshot(new_row)
  );

  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;
  RETURN NEW;
END;
$$;

DO $$
DECLARE
  audited_table text;
BEGIN
  FOREACH audited_table IN ARRAY ARRAY[
    'tenant_settings',
    'users',
    'roles',
    'user_roles',
    'role_permissions',
    'units',
    'categories',
    'packings',
    'plants',
    'suppliers',
    'raw_materials',
    'purchase_orders',
    'delivery_notes',
    'receiving_sessions',
    'receivings',
    'outgoing_sessions',
    'outgoing_documents',
    'inventory_ledger_entries',
    'kanban_lots'
  ]
  LOOP
    EXECUTE format('DROP TRIGGER IF EXISTS activity_audit ON %I', audited_table);
    EXECUTE format(
      'CREATE TRIGGER activity_audit AFTER INSERT OR UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION record_activity_change()',
      audited_table
    );
  END LOOP;
END;
$$;

REVOKE ALL ON activity_logs FROM PUBLIC;
GRANT SELECT ON activity_logs TO nextgen_app;

INSERT INTO permissions (code, description)
VALUES ('activity_log.view', 'View activity log')
ON CONFLICT (code) DO UPDATE SET description = EXCLUDED.description;

INSERT INTO role_permissions (tenant_id, role_id, permission_code)
SELECT tenant_id, role_id, 'activity_log.view'
FROM role_permissions
WHERE permission_code = 'configuration.manage'
ON CONFLICT DO NOTHING;

COMMIT;
