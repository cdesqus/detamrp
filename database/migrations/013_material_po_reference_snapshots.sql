ALTER TABLE raw_materials
  ADD COLUMN category_id uuid,
  ADD COLUMN packing_id uuid;

ALTER TABLE raw_materials
  ADD CONSTRAINT raw_materials_category_fk
    FOREIGN KEY (tenant_id, category_id) REFERENCES categories(tenant_id, id),
  ADD CONSTRAINT raw_materials_packing_fk
    FOREIGN KEY (tenant_id, packing_id) REFERENCES packings(tenant_id, id);

CREATE INDEX raw_materials_tenant_category_idx ON raw_materials (tenant_id, category_id);
CREATE INDEX raw_materials_tenant_packing_idx ON raw_materials (tenant_id, packing_id);

ALTER TABLE purchase_orders
  ADD COLUMN plant_id uuid,
  ADD COLUMN plant_code_snapshot text NOT NULL DEFAULT '',
  ADD COLUMN plant_name_snapshot text NOT NULL DEFAULT '',
  ADD COLUMN plant_address_snapshot text NOT NULL DEFAULT '';

ALTER TABLE purchase_orders
  ADD CONSTRAINT purchase_orders_plant_fk
    FOREIGN KEY (tenant_id, plant_id) REFERENCES plants(tenant_id, id);

CREATE INDEX purchase_orders_tenant_plant_idx ON purchase_orders (tenant_id, plant_id);

ALTER TABLE purchase_order_lines
  ADD COLUMN category_code_snapshot text NOT NULL DEFAULT '',
  ADD COLUMN category_name_snapshot text NOT NULL DEFAULT '',
  ADD COLUMN packing_code_snapshot text NOT NULL DEFAULT '',
  ADD COLUMN packing_name_snapshot text NOT NULL DEFAULT '';
