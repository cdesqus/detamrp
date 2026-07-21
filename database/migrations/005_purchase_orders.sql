CREATE TABLE IF NOT EXISTS purchase_order_number_sequences (
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  year_month char(6) NOT NULL CHECK (year_month ~ '^[0-9]{6}$'),
  next_value integer NOT NULL DEFAULT 1 CHECK (next_value > 0),
  PRIMARY KEY (tenant_id, year_month)
);

CREATE TABLE IF NOT EXISTS purchase_orders (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  po_number text NOT NULL,
  supplier_id uuid NOT NULL,
  order_date date NOT NULL,
  expected_delivery_date date NOT NULL,
  currency char(3) NOT NULL,
  notes text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PENDING_APPROVAL', 'APPROVED', 'REJECTED', 'CANCELLED')),
  version integer NOT NULL DEFAULT 0 CHECK (version >= 0),
  total_amount numeric(20,6) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
  sage_purchase_order_number text,
  submitted_approver_user_id uuid,
  submitted_approver_display_name text NOT NULL DEFAULT '',
  submitted_approver_email text NOT NULL DEFAULT '',
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_by_user_id uuid NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, po_number),
  CHECK (expected_delivery_date >= order_date),
  FOREIGN KEY (tenant_id, supplier_id) REFERENCES suppliers(tenant_id, id),
  FOREIGN KEY (tenant_id, submitted_approver_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, id)
);

CREATE TABLE IF NOT EXISTS purchase_order_lines (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  purchase_order_id uuid NOT NULL,
  raw_material_id uuid NOT NULL,
  raw_material_code_snapshot text NOT NULL,
  raw_material_name_snapshot text NOT NULL,
  base_unit_id uuid NOT NULL,
  base_unit_code_snapshot text NOT NULL,
  qty_per_kanban_snapshot numeric(20,6) NOT NULL CHECK (qty_per_kanban_snapshot > 0),
  total_kanban numeric(20,6) NOT NULL CHECK (total_kanban > 0 AND total_kanban = trunc(total_kanban)),
  ordered_base_qty numeric(20,6) NOT NULL CHECK (ordered_base_qty > 0),
  unit_price_snapshot numeric(20,6) NOT NULL CHECK (unit_price_snapshot >= 0),
  line_total numeric(20,6) NOT NULL CHECK (line_total >= 0),
  sort_position integer NOT NULL CHECK (sort_position > 0),
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_by_user_id uuid NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, purchase_order_id, raw_material_id),
  UNIQUE (tenant_id, purchase_order_id, sort_position),
  FOREIGN KEY (tenant_id, purchase_order_id) REFERENCES purchase_orders(tenant_id, id),
  FOREIGN KEY (tenant_id, raw_material_id) REFERENCES raw_materials(tenant_id, id),
  FOREIGN KEY (tenant_id, base_unit_id) REFERENCES measurements(tenant_id, id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, id)
);

CREATE TABLE IF NOT EXISTS purchase_order_approvals (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  purchase_order_id uuid NOT NULL,
  version integer NOT NULL CHECK (version > 0),
  approver_user_id uuid NOT NULL,
  approver_display_name text NOT NULL,
  approver_email text NOT NULL,
  status text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED')),
  decision_reason text NOT NULL DEFAULT '',
  decided_at timestamptz,
  decided_by_user_id uuid,
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_by_user_id uuid NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, purchase_order_id, version),
  CHECK ((status = 'PENDING' AND decision_reason = '' AND decided_at IS NULL AND decided_by_user_id IS NULL) OR
         (status = 'APPROVED' AND decided_at IS NOT NULL AND decided_by_user_id IS NOT NULL) OR
         (status = 'REJECTED' AND decision_reason <> '' AND decided_at IS NOT NULL AND decided_by_user_id IS NOT NULL)),
  FOREIGN KEY (tenant_id, purchase_order_id) REFERENCES purchase_orders(tenant_id, id),
  FOREIGN KEY (tenant_id, approver_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, decided_by_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, id)
);

CREATE OR REPLACE FUNCTION purchase_order_supplier_matches_material()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  order_supplier_id uuid;
  material_supplier_id uuid;
BEGIN
  SELECT supplier_id INTO order_supplier_id
  FROM purchase_orders
  WHERE tenant_id = NEW.tenant_id AND id = NEW.purchase_order_id;

  SELECT supplier_id INTO material_supplier_id
  FROM raw_materials
  WHERE tenant_id = NEW.tenant_id AND id = NEW.raw_material_id;

  IF order_supplier_id IS NULL OR material_supplier_id IS NULL OR order_supplier_id <> material_supplier_id THEN
    RAISE EXCEPTION 'purchase order raw material must belong to the order supplier';
  END IF;
  RETURN NEW;
END;
$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'purchase_order_line_supplier_check' AND tgrelid = 'purchase_order_lines'::regclass) THEN
    EXECUTE 'CREATE TRIGGER purchase_order_line_supplier_check
      BEFORE INSERT OR UPDATE OF tenant_id, purchase_order_id, raw_material_id ON purchase_order_lines
      FOR EACH ROW EXECUTE FUNCTION purchase_order_supplier_matches_material()';
  END IF;
END
$$;

CREATE INDEX IF NOT EXISTS purchase_orders_tenant_status_order_date_idx ON purchase_orders (tenant_id, status, order_date DESC);
CREATE INDEX IF NOT EXISTS purchase_orders_tenant_supplier_order_date_idx ON purchase_orders (tenant_id, supplier_id, order_date DESC);
CREATE INDEX IF NOT EXISTS purchase_order_lines_tenant_order_sort_idx ON purchase_order_lines (tenant_id, purchase_order_id, sort_position);
CREATE INDEX IF NOT EXISTS purchase_order_approvals_tenant_approver_status_idx ON purchase_order_approvals (tenant_id, approver_user_id, status, created_at DESC);

ALTER TABLE purchase_order_number_sequences ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_order_number_sequences FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'purchase_order_number_sequences' AND policyname = 'purchase_order_number_sequences_isolation') THEN
    EXECUTE 'CREATE POLICY purchase_order_number_sequences_isolation ON purchase_order_number_sequences
      USING (tenant_id = current_setting(''app.tenant_id'', true)::uuid)
      WITH CHECK (tenant_id = current_setting(''app.tenant_id'', true)::uuid)';
  END IF;
END $$;

ALTER TABLE purchase_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_orders FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'purchase_orders' AND policyname = 'purchase_orders_isolation') THEN
    EXECUTE 'CREATE POLICY purchase_orders_isolation ON purchase_orders
      USING (tenant_id = current_setting(''app.tenant_id'', true)::uuid)
      WITH CHECK (tenant_id = current_setting(''app.tenant_id'', true)::uuid)';
  END IF;
END $$;

ALTER TABLE purchase_order_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_order_lines FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'purchase_order_lines' AND policyname = 'purchase_order_lines_isolation') THEN
    EXECUTE 'CREATE POLICY purchase_order_lines_isolation ON purchase_order_lines
      USING (tenant_id = current_setting(''app.tenant_id'', true)::uuid)
      WITH CHECK (tenant_id = current_setting(''app.tenant_id'', true)::uuid)';
  END IF;
END $$;

ALTER TABLE purchase_order_approvals ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_order_approvals FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'purchase_order_approvals' AND policyname = 'purchase_order_approvals_isolation') THEN
    EXECUTE 'CREATE POLICY purchase_order_approvals_isolation ON purchase_order_approvals
      USING (tenant_id = current_setting(''app.tenant_id'', true)::uuid)
      WITH CHECK (tenant_id = current_setting(''app.tenant_id'', true)::uuid)';
  END IF;
END $$;

GRANT SELECT, INSERT, UPDATE ON purchase_order_number_sequences, purchase_orders, purchase_order_approvals TO nextgen_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON purchase_order_lines TO nextgen_app;
