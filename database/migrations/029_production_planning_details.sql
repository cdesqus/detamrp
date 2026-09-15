BEGIN;
CREATE TABLE production_plan_counters (
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 period text NOT NULL,
 last_number bigint NOT NULL CHECK (last_number > 0),
 PRIMARY KEY(tenant_id,period)
);
ALTER TABLE production_plan_counters ENABLE ROW LEVEL SECURITY;
ALTER TABLE production_plan_counters FORCE ROW LEVEL SECURITY;
CREATE POLICY production_plan_counters_isolation ON production_plan_counters USING (tenant_id=current_setting('app.tenant_id',true)::uuid) WITH CHECK (tenant_id=current_setting('app.tenant_id',true)::uuid);
GRANT SELECT,INSERT,UPDATE ON production_plan_counters TO nextgen_app;
ALTER TABLE production_plan_lines ADD COLUMN part_number text NOT NULL DEFAULT '', ADD COLUMN part_name text NOT NULL DEFAULT '';
UPDATE production_plan_lines l SET part_number=f.item_code,part_name=f.name FROM finished_goods f WHERE f.tenant_id=l.tenant_id AND f.id=l.finished_good_id;
UPDATE production_plan_lines l SET part_number=r.code,part_name=r.name FROM raw_materials r WHERE r.tenant_id=l.tenant_id AND r.id=l.raw_material_id;
CREATE INDEX production_plans_updated_idx ON production_plans(tenant_id,updated_at DESC);
CREATE INDEX production_plan_lines_plan_idx ON production_plan_lines(tenant_id,plan_id);
CREATE INDEX production_orders_plan_idx ON production_orders(tenant_id,plan_id);
CREATE TRIGGER activity_audit AFTER INSERT OR UPDATE OR DELETE ON production_plans FOR EACH ROW EXECUTE FUNCTION record_activity_change();
CREATE TRIGGER activity_audit AFTER INSERT OR UPDATE OR DELETE ON production_plan_lines FOR EACH ROW EXECUTE FUNCTION record_activity_change();
CREATE TRIGGER activity_audit AFTER INSERT OR UPDATE OR DELETE ON production_orders FOR EACH ROW EXECUTE FUNCTION record_activity_change();
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
    WHEN 'production_plans' THEN 'PRODUCTION'
    WHEN 'production_plan_lines' THEN 'PRODUCTION'
    WHEN 'production_orders' THEN 'PRODUCTION'
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
    source_row->>'plan_number',
    source_row->>'order_number',
    source_row->>'part_number',
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
      WHEN 'CLOSED' THEN 'CLOSED'
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
COMMIT;

