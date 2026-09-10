BEGIN;

CREATE TABLE boms (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  output_kind text NOT NULL CHECK (output_kind IN ('FG','RAW_MATERIAL')),
  finished_good_id uuid,
  raw_material_id uuid,
  revision integer NOT NULL CHECK (revision > 0),
  status text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','ACTIVE','REVISED')),
  notes text NOT NULL DEFAULT '',
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_by_user_id uuid NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id,id),
  UNIQUE (tenant_id,output_kind,finished_good_id,raw_material_id,revision),
  CHECK ((output_kind='FG' AND finished_good_id IS NOT NULL AND raw_material_id IS NULL) OR (output_kind='RAW_MATERIAL' AND raw_material_id IS NOT NULL AND finished_good_id IS NULL)),
  FOREIGN KEY (tenant_id,finished_good_id) REFERENCES finished_goods(tenant_id,id),
  FOREIGN KEY (tenant_id,raw_material_id) REFERENCES raw_materials(tenant_id,id),
  FOREIGN KEY (tenant_id,created_by_user_id) REFERENCES users(tenant_id,id),
  FOREIGN KEY (tenant_id,updated_by_user_id) REFERENCES users(tenant_id,id)
);
CREATE UNIQUE INDEX boms_one_active_finished_good ON boms(tenant_id,finished_good_id) WHERE status='ACTIVE';
CREATE UNIQUE INDEX boms_one_active_raw_material ON boms(tenant_id,raw_material_id) WHERE status='ACTIVE';

CREATE TABLE bom_components (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  bom_id uuid NOT NULL,
  raw_material_id uuid NOT NULL,
  usage_qty numeric(20,6) NOT NULL CHECK (usage_qty>0),
  sort_position integer NOT NULL CHECK (sort_position>=0),
  UNIQUE (tenant_id,id), UNIQUE (tenant_id,bom_id,raw_material_id),
  FOREIGN KEY (tenant_id,bom_id) REFERENCES boms(tenant_id,id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id,raw_material_id) REFERENCES raw_materials(tenant_id,id)
);
ALTER TABLE boms ENABLE ROW LEVEL SECURITY; ALTER TABLE boms FORCE ROW LEVEL SECURITY;
CREATE POLICY boms_isolation ON boms USING (tenant_id=current_setting('app.tenant_id',true)::uuid) WITH CHECK (tenant_id=current_setting('app.tenant_id',true)::uuid);
ALTER TABLE bom_components ENABLE ROW LEVEL SECURITY; ALTER TABLE bom_components FORCE ROW LEVEL SECURITY;
CREATE POLICY bom_components_isolation ON bom_components USING (tenant_id=current_setting('app.tenant_id',true)::uuid) WITH CHECK (tenant_id=current_setting('app.tenant_id',true)::uuid);
GRANT SELECT,INSERT,UPDATE,DELETE ON boms,bom_components TO nextgen_app;
INSERT INTO permissions(code,description) VALUES ('bom.view','View BOMs'),('bom.manage','Manage BOM drafts'),('bom.activate','Activate BOMs') ON CONFLICT(code) DO UPDATE SET description=EXCLUDED.description;
INSERT INTO role_permissions(tenant_id,role_id,permission_code) SELECT r.tenant_id,r.id,p.code FROM roles r CROSS JOIN permissions p WHERE r.code='ADMIN' AND p.code IN ('bom.view','bom.manage','bom.activate') ON CONFLICT DO NOTHING;
CREATE TRIGGER activity_audit AFTER INSERT OR UPDATE OR DELETE ON boms FOR EACH ROW EXECUTE FUNCTION record_activity_change();
COMMIT;
