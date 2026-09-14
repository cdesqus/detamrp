BEGIN;

-- Production planning is the source document for manufacturing activity.
CREATE TABLE production_plans (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  plan_number text NOT NULL,
  period_start date NOT NULL,
  period_end date NOT NULL,
  plant_id uuid REFERENCES plants(id),
  status text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','APPROVED','CLOSED')),
  notes text NOT NULL DEFAULT '',
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_by_user_id uuid NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, plan_number),
  CHECK (period_end >= period_start),
  FOREIGN KEY (tenant_id, plant_id) REFERENCES plants(tenant_id, id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, id)
);

CREATE TABLE production_plan_lines (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  plan_id uuid NOT NULL,
  finished_good_id uuid,
  raw_material_id uuid,
  planned_qty numeric(20,6) NOT NULL CHECK (planned_qty > 0),
  unit_code text NOT NULL,
  sort_position integer NOT NULL DEFAULT 0,
  UNIQUE (tenant_id, id),
  CHECK ((finished_good_id IS NOT NULL) <> (raw_material_id IS NOT NULL)),
  FOREIGN KEY (tenant_id, plan_id) REFERENCES production_plans(tenant_id, id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, finished_good_id) REFERENCES finished_goods(tenant_id, id),
  FOREIGN KEY (tenant_id, raw_material_id) REFERENCES raw_materials(tenant_id, id)
);

CREATE TABLE production_orders (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  order_number text NOT NULL,
  plan_id uuid NOT NULL,
  finished_good_id uuid,
  raw_material_id uuid,
  planned_qty numeric(20,6) NOT NULL CHECK (planned_qty > 0),
  bom_revision integer,
  bom_snapshot jsonb,
  status text NOT NULL DEFAULT 'RELEASED' CHECK (status IN ('RELEASED','IN_PROGRESS','PARTIAL','COMPLETED','CANCELLED')),
  notes text NOT NULL DEFAULT '',
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, order_number),
  CHECK ((finished_good_id IS NOT NULL) <> (raw_material_id IS NOT NULL)),
  FOREIGN KEY (tenant_id, plan_id) REFERENCES production_plans(tenant_id, id),
  FOREIGN KEY (tenant_id, finished_good_id) REFERENCES finished_goods(tenant_id, id),
  FOREIGN KEY (tenant_id, raw_material_id) REFERENCES raw_materials(tenant_id, id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id)
);

CREATE TABLE production_operations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  production_order_id uuid NOT NULL,
  operation_code text NOT NULL CHECK (operation_code <> ''),
  operation_name text NOT NULL,
  sequence_no integer NOT NULL CHECK (sequence_no > 0),
  planned_qty numeric(20,6) NOT NULL CHECK (planned_qty > 0),
  completed_qty numeric(20,6) NOT NULL DEFAULT 0 CHECK (completed_qty >= 0),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, production_order_id, sequence_no),
  FOREIGN KEY (tenant_id, production_order_id) REFERENCES production_orders(tenant_id, id) ON DELETE CASCADE,
  CHECK (completed_qty <= planned_qty)
);

CREATE TABLE production_entries (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  operation_id uuid NOT NULL,
  production_date date NOT NULL,
  shift_code text NOT NULL DEFAULT '',
  qty_processed numeric(20,6) NOT NULL CHECK (qty_processed > 0),
  qty_good numeric(20,6) NOT NULL CHECK (qty_good >= 0),
  qty_rejected numeric(20,6) NOT NULL DEFAULT 0 CHECK (qty_rejected >= 0),
  material_cost numeric(20,6) NOT NULL DEFAULT 0 CHECK (material_cost >= 0),
  process_cost numeric(20,6) NOT NULL DEFAULT 0 CHECK (process_cost >= 0),
  notes text NOT NULL DEFAULT '',
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  FOREIGN KEY (tenant_id, operation_id) REFERENCES production_operations(tenant_id, id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id),
  CHECK (qty_good + qty_rejected <= qty_processed)
);

ALTER TABLE production_plans ENABLE ROW LEVEL SECURITY; ALTER TABLE production_plans FORCE ROW LEVEL SECURITY;
ALTER TABLE production_plan_lines ENABLE ROW LEVEL SECURITY; ALTER TABLE production_plan_lines FORCE ROW LEVEL SECURITY;
ALTER TABLE production_orders ENABLE ROW LEVEL SECURITY; ALTER TABLE production_orders FORCE ROW LEVEL SECURITY;
ALTER TABLE production_operations ENABLE ROW LEVEL SECURITY; ALTER TABLE production_operations FORCE ROW LEVEL SECURITY;
ALTER TABLE production_entries ENABLE ROW LEVEL SECURITY; ALTER TABLE production_entries FORCE ROW LEVEL SECURITY;

CREATE POLICY production_plans_isolation ON production_plans USING (tenant_id=current_setting('app.tenant_id',true)::uuid) WITH CHECK (tenant_id=current_setting('app.tenant_id',true)::uuid);
CREATE POLICY production_plan_lines_isolation ON production_plan_lines USING (tenant_id=current_setting('app.tenant_id',true)::uuid) WITH CHECK (tenant_id=current_setting('app.tenant_id',true)::uuid);
CREATE POLICY production_orders_isolation ON production_orders USING (tenant_id=current_setting('app.tenant_id',true)::uuid) WITH CHECK (tenant_id=current_setting('app.tenant_id',true)::uuid);
CREATE POLICY production_operations_isolation ON production_operations USING (tenant_id=current_setting('app.tenant_id',true)::uuid) WITH CHECK (tenant_id=current_setting('app.tenant_id',true)::uuid);
CREATE POLICY production_entries_isolation ON production_entries USING (tenant_id=current_setting('app.tenant_id',true)::uuid) WITH CHECK (tenant_id=current_setting('app.tenant_id',true)::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON production_plans, production_plan_lines, production_orders, production_operations, production_entries TO nextgen_app;
INSERT INTO permissions(code, description) VALUES
 ('production.view','View production planning and execution'),
 ('production.plan','Create and approve production plans'),
 ('production.order','Create and manage production orders'),
 ('production.entry','Record daily production')
 ON CONFLICT(code) DO UPDATE SET description=EXCLUDED.description;
INSERT INTO role_permissions(tenant_id, role_id, permission_code)
 SELECT r.tenant_id, r.id, p.code FROM roles r CROSS JOIN permissions p
 WHERE r.code='ADMIN' AND p.code LIKE 'production.%' ON CONFLICT DO NOTHING;

COMMIT;
