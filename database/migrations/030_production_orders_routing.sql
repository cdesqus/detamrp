BEGIN;
CREATE TABLE production_routings (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id),
 finished_good_id uuid, raw_material_id uuid, name text NOT NULL, revision integer NOT NULL DEFAULT 1,
 currency text NOT NULL DEFAULT 'IDR', steps jsonb NOT NULL, active boolean NOT NULL DEFAULT true,
 created_by_user_id uuid NOT NULL, updated_by_user_id uuid NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(tenant_id,id), CHECK ((finished_good_id IS NOT NULL) <> (raw_material_id IS NOT NULL)),
 CHECK (jsonb_typeof(steps)='array' AND jsonb_array_length(steps)>0),
 FOREIGN KEY(tenant_id,finished_good_id) REFERENCES finished_goods(tenant_id,id),
 FOREIGN KEY(tenant_id,raw_material_id) REFERENCES raw_materials(tenant_id,id),
 FOREIGN KEY(tenant_id,created_by_user_id) REFERENCES users(tenant_id,id),
 FOREIGN KEY(tenant_id,updated_by_user_id) REFERENCES users(tenant_id,id)
);
CREATE UNIQUE INDEX production_routings_active_fg ON production_routings(tenant_id,finished_good_id) WHERE active;
CREATE UNIQUE INDEX production_routings_active_rm ON production_routings(tenant_id,raw_material_id) WHERE active;
CREATE UNIQUE INDEX production_routings_revision_fg ON production_routings(tenant_id,finished_good_id,revision) WHERE finished_good_id IS NOT NULL;
CREATE UNIQUE INDEX production_routings_revision_rm ON production_routings(tenant_id,raw_material_id,revision) WHERE raw_material_id IS NOT NULL;
ALTER TABLE production_routings ENABLE ROW LEVEL SECURITY; ALTER TABLE production_routings FORCE ROW LEVEL SECURITY;
CREATE POLICY production_routings_isolation ON production_routings USING(tenant_id=current_setting('app.tenant_id',true)::uuid) WITH CHECK(tenant_id=current_setting('app.tenant_id',true)::uuid);
GRANT SELECT,INSERT,UPDATE,DELETE ON production_routings TO nextgen_app;
CREATE TRIGGER activity_audit AFTER INSERT OR UPDATE OR DELETE ON production_routings FOR EACH ROW EXECUTE FUNCTION record_activity_change();

ALTER TABLE production_orders ADD COLUMN plan_line_id uuid, ADD COLUMN routing_id uuid, ADD COLUMN routing_revision integer,
 ADD COLUMN part_number text NOT NULL DEFAULT '', ADD COLUMN part_name text NOT NULL DEFAULT '', ADD COLUMN unit_code text NOT NULL DEFAULT '',
 ADD COLUMN plant_id uuid, ADD COLUMN period_start date, ADD COLUMN due_date date,
 ADD COLUMN currency text NOT NULL DEFAULT 'IDR', ADD COLUMN material_estimate numeric(20,6) NOT NULL DEFAULT 0,
 ADD COLUMN process_estimate numeric(20,6) NOT NULL DEFAULT 0, ADD COLUMN cancel_reason text NOT NULL DEFAULT '', ADD COLUMN updated_by_user_id uuid;
UPDATE production_orders o SET plan_line_id=(SELECT l.id FROM production_plan_lines l WHERE l.tenant_id=o.tenant_id AND l.plan_id=o.plan_id AND (l.finished_good_id=o.finished_good_id OR l.raw_material_id=o.raw_material_id) ORDER BY l.sort_position LIMIT 1),plant_id=p.plant_id,period_start=p.period_start,due_date=p.period_end,updated_by_user_id=o.created_by_user_id FROM production_plans p WHERE p.tenant_id=o.tenant_id AND p.id=o.plan_id;
UPDATE production_orders o SET part_number=l.part_number,part_name=l.part_name,unit_code=l.unit_code FROM production_plan_lines l WHERE l.tenant_id=o.tenant_id AND l.id=o.plan_line_id;
ALTER TABLE production_orders ADD FOREIGN KEY(tenant_id,plan_line_id) REFERENCES production_plan_lines(tenant_id,id),
 ADD FOREIGN KEY(tenant_id,routing_id) REFERENCES production_routings(tenant_id,id),
 ADD FOREIGN KEY(tenant_id,plant_id) REFERENCES plants(tenant_id,id),
 ADD FOREIGN KEY(tenant_id,updated_by_user_id) REFERENCES users(tenant_id,id),
 ADD CHECK (due_date>=period_start);
ALTER TABLE production_operations ADD COLUMN rate_snapshot numeric(20,6) NOT NULL DEFAULT 0 CHECK(rate_snapshot>=0);
ALTER TABLE production_entries ADD COLUMN voided_at timestamptz;
CREATE INDEX production_orders_line_idx ON production_orders(tenant_id,plan_line_id);
INSERT INTO permissions(code,description) VALUES ('production.routing','Manage production routings'),('production.void','Void production transactions'),('production.period','Close production periods'),('production.report','View production costs and export reports'),('production.wip','Transfer work in progress') ON CONFLICT(code) DO UPDATE SET description=EXCLUDED.description;
INSERT INTO role_permissions(tenant_id,role_id,permission_code) SELECT r.tenant_id,r.id,p.code FROM roles r CROSS JOIN permissions p WHERE r.code='ADMIN' AND p.code LIKE 'production.%' ON CONFLICT DO NOTHING;
COMMIT;
