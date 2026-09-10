-- One-time demo data for the default live tenant.
-- Run explicitly; this is intentionally not part of the migration runner.
-- psql -U nextgen_app -d nextgen -f database/seed-live-demo.sql
BEGIN;
SET LOCAL app.tenant_id = '00000000-0000-0000-0000-000000000001';

DO $$
DECLARE
  v_tenant uuid := '00000000-0000-0000-0000-000000000001';
  v_user uuid;
  v_unit uuid;
  v_supplier uuid := 'd0000000-0000-4000-8000-000000000001';
  v_rm1 uuid := 'd0000000-0000-4000-8000-000000000011';
  v_rm2 uuid := 'd0000000-0000-4000-8000-000000000012';
  v_rm3 uuid := 'd0000000-0000-4000-8000-000000000013';
  v_fg1 uuid := 'd0000000-0000-4000-8000-000000000021';
  v_fg2 uuid := 'd0000000-0000-4000-8000-000000000022';
  v_bom1 uuid := 'd0000000-0000-4000-8000-000000000031';
  v_bom2 uuid := 'd0000000-0000-4000-8000-000000000032';
BEGIN
  SELECT id INTO v_user FROM users WHERE tenant_id = v_tenant ORDER BY created_at LIMIT 1;
  IF v_user IS NULL THEN RAISE EXCEPTION 'No user exists for demo tenant %', v_tenant; END IF;

  SELECT id INTO v_unit FROM units WHERE tenant_id = v_tenant AND code = 'PC' LIMIT 1;
  IF v_unit IS NULL THEN
    v_unit := 'd0000000-0000-4000-8000-000000000002';
    INSERT INTO units (id,tenant_id,code,name,decimal_allowed,active,created_by_user_id,updated_by_user_id)
    VALUES (v_unit,v_tenant,'PC','Piece',false,true,v_user,v_user)
    ON CONFLICT (tenant_id,id) DO UPDATE SET active=true,updated_at=now(),updated_by_user_id=EXCLUDED.updated_by_user_id;
  END IF;

  INSERT INTO suppliers (id,tenant_id,code,sage_supplier_code,name,email,phone,address,contact_person,currency,active,created_by_user_id,updated_by_user_id)
  VALUES (v_supplier,v_tenant,'DEMO-SUPPLIER','DEMO-SUPPLIER','Demo Supplier','demo@example.invalid','','','Demo Contact','IDR',true,v_user,v_user)
  ON CONFLICT (tenant_id,id) DO UPDATE SET active=true,updated_at=now(),updated_by_user_id=EXCLUDED.updated_by_user_id;

  INSERT INTO raw_materials (id,tenant_id,code,sage_item_code,name,supplier_id,base_unit_id,qty_per_kanban,minimum_stock,description,standard_unit_price,currency,active,created_by_user_id,updated_by_user_id)
  VALUES
    (v_rm1,v_tenant,'DEMO-STEEL','DEMO-STEEL','Demo Steel Plate',v_supplier,v_unit,20,0,'Demo BOM material',15000,'IDR',true,v_user,v_user),
    (v_rm2,v_tenant,'DEMO-BOLT','DEMO-BOLT','Demo Hex Bolt',v_supplier,v_unit,100,0,'Demo BOM material',2500,'IDR',true,v_user,v_user),
    (v_rm3,v_tenant,'DEMO-RUBBER','DEMO-RUBBER','Demo Rubber Pad',v_supplier,v_unit,50,0,'Demo BOM material',5000,'IDR',true,v_user,v_user)
  ON CONFLICT (tenant_id,id) DO UPDATE SET active=true,updated_at=now(),updated_by_user_id=EXCLUDED.updated_by_user_id;

  INSERT INTO finished_goods (id,tenant_id,item_code,name,base_unit_id,sales_price,currency,price_version,active,created_by_user_id,updated_by_user_id)
  VALUES
    (v_fg1,v_tenant,'DEMO-ASSEMBLY','Demo Assembly',v_unit,250000,'IDR',1,true,v_user,v_user),
    (v_fg2,v_tenant,'DEMO-SUBASSEMBLY','Demo Sub Assembly',v_unit,125000,'IDR',1,true,v_user,v_user)
  ON CONFLICT (tenant_id,id) DO UPDATE SET active=true,updated_at=now(),updated_by_user_id=EXCLUDED.updated_by_user_id;

  INSERT INTO boms (id,tenant_id,output_kind,finished_good_id,revision,status,notes,created_by_user_id,updated_by_user_id)
  VALUES
    (v_bom1,v_tenant,'FG',v_fg1,1,'ACTIVE','Demo recipe',v_user,v_user),
    (v_bom2,v_tenant,'FG',v_fg2,1,'ACTIVE','Demo recipe',v_user,v_user)
  ON CONFLICT (tenant_id,id) DO UPDATE SET status='ACTIVE',notes=EXCLUDED.notes,updated_at=now(),updated_by_user_id=EXCLUDED.updated_by_user_id;

  INSERT INTO bom_components (tenant_id,bom_id,raw_material_id,usage_qty,sort_position)
  VALUES
    (v_tenant,v_bom1,v_rm1,2,0),(v_tenant,v_bom1,v_rm2,4,1),
    (v_tenant,v_bom2,v_rm1,1,0),(v_tenant,v_bom2,v_rm3,2,1)
  ON CONFLICT (tenant_id,bom_id,raw_material_id) DO UPDATE SET usage_qty=EXCLUDED.usage_qty,sort_position=EXCLUDED.sort_position;

  -- Additional demo items for list, filtering, and revision testing.
  FOR i IN 4..12 LOOP
    INSERT INTO raw_materials (tenant_id,code,sage_item_code,name,supplier_id,base_unit_id,qty_per_kanban,minimum_stock,description,standard_unit_price,currency,active,created_by_user_id,updated_by_user_id)
    VALUES (v_tenant,format('DEMO-RM-%s',i),format('DEMO-RM-%s',i),format('Demo Material %s',i),v_supplier,v_unit,10+i,0,'Demo BOM material',1000*i,'IDR',true,v_user,v_user)
    ON CONFLICT (tenant_id,code) DO UPDATE SET active=true,updated_at=now(),updated_by_user_id=EXCLUDED.updated_by_user_id;
  END LOOP;
  FOR i IN 3..8 LOOP
    INSERT INTO finished_goods (tenant_id,item_code,name,base_unit_id,sales_price,currency,price_version,active,created_by_user_id,updated_by_user_id)
    VALUES (v_tenant,format('DEMO-FG-%s',i),format('Demo Finished Good %s',i),v_unit,100000*i,'IDR',1,true,v_user,v_user)
    ON CONFLICT (tenant_id,item_code) DO UPDATE SET active=true,updated_at=now(),updated_by_user_id=EXCLUDED.updated_by_user_id;
    SELECT id INTO v_fg1 FROM finished_goods WHERE tenant_id=v_tenant AND item_code=format('DEMO-FG-%s',i);
    SELECT id INTO v_rm1 FROM raw_materials WHERE tenant_id=v_tenant AND code=format('DEMO-RM-%s',i+1);
    SELECT id INTO v_rm2 FROM raw_materials WHERE tenant_id=v_tenant AND code=format('DEMO-RM-%s',i+2);
    SELECT id INTO v_bom1 FROM boms WHERE tenant_id=v_tenant AND output_kind='FG' AND finished_good_id=v_fg1 AND revision=1;
    IF v_bom1 IS NULL THEN
      INSERT INTO boms (tenant_id,output_kind,finished_good_id,revision,status,notes,created_by_user_id,updated_by_user_id)
      VALUES (v_tenant,'FG',v_fg1,1,'ACTIVE','Demo recipe',v_user,v_user) RETURNING id INTO v_bom1;
    ELSE
      UPDATE boms SET status='ACTIVE',updated_at=now(),updated_by_user_id=v_user WHERE id=v_bom1;
    END IF;
    INSERT INTO bom_components (tenant_id,bom_id,raw_material_id,usage_qty,sort_position)
    VALUES (v_tenant,v_bom1,v_rm1,1+i/10.0,0),(v_tenant,v_bom1,v_rm2,2,1)
    ON CONFLICT (tenant_id,bom_id,raw_material_id) DO UPDATE SET usage_qty=EXCLUDED.usage_qty,sort_position=EXCLUDED.sort_position;
  END LOOP;
END $$;
COMMIT;

