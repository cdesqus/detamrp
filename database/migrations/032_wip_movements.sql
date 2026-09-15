BEGIN;
-- Append-only work-in-progress ledger. Every row is a signed movement so a
-- balance is always the sum of its movements and stays reconcilable.
CREATE TABLE production_wip_movements (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  movement_sequence bigint GENERATED ALWAYS AS IDENTITY,
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  production_order_id uuid NOT NULL,
  movement_type text NOT NULL CHECK (movement_type IN ('RECEIPT','TRANSFER','CONSUMPTION')),
  source_operation_id uuid,
  destination_operation_id uuid,
  quantity numeric(20,6) NOT NULL CHECK (quantity <> 0),
  unit_cost numeric(20,6) NOT NULL DEFAULT 0 CHECK (unit_cost >= 0),
  total_cost numeric(20,6) NOT NULL DEFAULT 0,
  currency text NOT NULL DEFAULT 'IDR',
  entry_id uuid,
  lot_movement_id uuid,
  reverses_id uuid,
  movement_date date NOT NULL,
  notes text NOT NULL DEFAULT '',
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id,id),
  FOREIGN KEY (tenant_id,production_order_id) REFERENCES production_orders(tenant_id,id),
  FOREIGN KEY (tenant_id,source_operation_id) REFERENCES production_operations(tenant_id,id),
  FOREIGN KEY (tenant_id,destination_operation_id) REFERENCES production_operations(tenant_id,id),
  FOREIGN KEY (tenant_id,entry_id) REFERENCES production_entries(tenant_id,id),
  FOREIGN KEY (tenant_id,lot_movement_id) REFERENCES production_wip_movements(tenant_id,id),
  FOREIGN KEY (tenant_id,reverses_id) REFERENCES production_wip_movements(tenant_id,id),
  FOREIGN KEY (tenant_id,created_by_user_id) REFERENCES users(tenant_id,id),
  -- RECEIPT lands on an operation, CONSUMPTION leaves one, TRANSFER does both.
  CHECK ((movement_type='RECEIPT' AND source_operation_id IS NULL AND destination_operation_id IS NOT NULL)
      OR (movement_type='CONSUMPTION' AND destination_operation_id IS NULL AND source_operation_id IS NOT NULL)
      OR (movement_type='TRANSFER' AND source_operation_id IS NOT NULL AND destination_operation_id IS NOT NULL AND source_operation_id<>destination_operation_id)),
  -- Only a reversal may carry a negative quantity.
  CHECK (quantity > 0 OR reverses_id IS NOT NULL)
);
CREATE INDEX production_wip_order_idx ON production_wip_movements(tenant_id,production_order_id,movement_date,created_at);
CREATE INDEX production_wip_lot_idx ON production_wip_movements(tenant_id,lot_movement_id);
CREATE INDEX production_wip_entry_idx ON production_wip_movements(tenant_id,entry_id);
CREATE UNIQUE INDEX production_wip_single_reversal ON production_wip_movements(tenant_id,reverses_id) WHERE reverses_id IS NOT NULL;

ALTER TABLE production_wip_movements ENABLE ROW LEVEL SECURITY;
ALTER TABLE production_wip_movements FORCE ROW LEVEL SECURITY;
CREATE POLICY production_wip_movements_isolation ON production_wip_movements USING (tenant_id=current_setting('app.tenant_id',true)::uuid) WITH CHECK (tenant_id=current_setting('app.tenant_id',true)::uuid);
GRANT SELECT,INSERT ON production_wip_movements TO nextgen_app;

-- The ledger is history: rows may never be updated or deleted, only reversed.
CREATE FUNCTION guard_wip_ledger() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'WIP movements are append-only; post a reversal instead' USING ERRCODE='23514';
END; $$;
CREATE TRIGGER production_wip_append_only BEFORE UPDATE OR DELETE ON production_wip_movements FOR EACH ROW EXECUTE FUNCTION guard_wip_ledger();

CREATE FUNCTION audit_wip_movement() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=public AS $$
DECLARE actor_id uuid; actor_name text; order_number text;
BEGIN
  actor_id := NULLIF(current_setting('app.user_id',true),'')::uuid;
  SELECT COALESCE(NULLIF(display_name,''),username) INTO actor_name FROM users WHERE tenant_id=NEW.tenant_id AND id=COALESCE(actor_id,NEW.created_by_user_id);
  SELECT o.order_number INTO order_number FROM production_orders o WHERE o.tenant_id=NEW.tenant_id AND o.id=NEW.production_order_id;
  INSERT INTO activity_logs(tenant_id,actor_user_id,actor_name,module,action,target_type,target_id,target_code,after_data)
  VALUES(NEW.tenant_id,actor_id,COALESCE(actor_name,'System'),'PRODUCTION',
    CASE WHEN NEW.reverses_id IS NOT NULL THEN 'WIP_REVERSED' ELSE 'WIP_'||NEW.movement_type END,
    'production_wip_movements',NEW.id,COALESCE(order_number,''),to_jsonb(NEW));
  RETURN NEW;
END; $$;
CREATE TRIGGER activity_audit AFTER INSERT ON production_wip_movements FOR EACH ROW EXECUTE FUNCTION audit_wip_movement();
COMMIT;
