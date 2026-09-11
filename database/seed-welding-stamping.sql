-- Sample part-number chain for stamping and welding processes.
-- Process outputs are represented as raw-material part numbers because the
-- current BOM model supports recursive RAW_MATERIAL BOMs.
-- Run explicitly after the normal master-data seed:
--   docker compose exec -T postgres psql -U nextgen_app -d nextgen < database/seed-welding-stamping.sql
BEGIN;
SET LOCAL app.tenant_id = '00000000-0000-0000-0000-000000000001';

DO $$
DECLARE
  v_tenant uuid := '00000000-0000-0000-0000-000000000001';
  v_user uuid;
  v_unit uuid;
  v_supplier uuid;
  v_sheet uuid := 'd0000000-0000-4000-8000-000000000101';
  v_tube uuid := 'd0000000-0000-4000-8000-000000000102';
  v_wire uuid := 'd0000000-0000-4000-8000-000000000103';
  v_gas uuid := 'd0000000-0000-4000-8000-000000000104';
  v_lube uuid := 'd0000000-0000-4000-8000-000000000105';
  v_stp_base uuid := 'd0000000-0000-4000-8000-000000000111';
  v_stp_support uuid := 'd0000000-0000-4000-8000-000000000112';
  v_wld_bracket uuid := 'd0000000-0000-4000-8000-000000000121';
  v_wld_frame uuid := 'd0000000-0000-4000-8000-000000000122';
  v_fg_bracket uuid := 'd0000000-0000-4000-8000-000000000131';
  v_fg_frame uuid := 'd0000000-0000-4000-8000-000000000132';
  v_bom_stp_base uuid := 'd0000000-0000-4000-8000-000000000201';
  v_bom_stp_support uuid := 'd0000000-0000-4000-8000-000000000202';
  v_bom_wld_bracket uuid := 'd0000000-0000-4000-8000-000000000203';
  v_bom_wld_frame uuid := 'd0000000-0000-4000-8000-000000000204';
  v_bom_fg_bracket uuid := 'd0000000-0000-4000-8000-000000000205';
  v_bom_fg_frame uuid := 'd0000000-0000-4000-8000-000000000206';
BEGIN
  SELECT id INTO v_user FROM users WHERE tenant_id=v_tenant ORDER BY created_at LIMIT 1;
  SELECT id INTO v_unit FROM units WHERE tenant_id=v_tenant AND code='PC' LIMIT 1;
  SELECT id INTO v_supplier FROM suppliers WHERE tenant_id=v_tenant AND code='NIS-001' LIMIT 1;
  IF v_user IS NULL OR v_unit IS NULL OR v_supplier IS NULL THEN
    RAISE EXCEPTION 'Run the standard sample master-data seed before welding/stamping seed';
  END IF;

  INSERT INTO raw_materials (id,tenant_id,code,sage_item_code,name,supplier_id,base_unit_id,qty_per_kanban,minimum_stock,description,standard_unit_price,currency,active,created_by_user_id,updated_by_user_id)
  VALUES
    (v_sheet,v_tenant,'RM-STEEL-SHEET','RM-STEEL-SHEET','Cold Rolled Steel Sheet',v_supplier,v_unit,20,0,'Sheet metal for stamping operations',18000,'IDR',true,v_user,v_user),
    (v_tube,v_tenant,'RM-STEEL-TUBE','RM-STEEL-TUBE','Mild Steel Tube',v_supplier,v_unit,30,0,'Structural tube for welded assemblies',22000,'IDR',true,v_user,v_user),
    (v_wire,v_tenant,'RM-WELD-WIRE','RM-WELD-WIRE','ER70S-6 Welding Wire',v_supplier,v_unit,10,0,'MIG welding consumable',85000,'IDR',true,v_user,v_user),
    (v_gas,v_tenant,'RM-SHIELD-GAS','RM-SHIELD-GAS','CO2 Shielding Gas',v_supplier,v_unit,5,0,'Shielding gas for welding operations',120000,'IDR',true,v_user,v_user),
    (v_lube,v_tenant,'RM-STAMP-LUBE','RM-STAMP-LUBE','Press Forming Lubricant',v_supplier,v_unit,10,0,'Lubricant for stamping dies',45000,'IDR',true,v_user,v_user),
    (v_stp_base,v_tenant,'STP-BASE-PLATE','STP-BASE-PLATE','Stamped Base Plate',v_supplier,v_unit,20,0,'Part number output from stamping operation',65000,'IDR',true,v_user,v_user),
    (v_stp_support,v_tenant,'STP-SUPPORT-PLATE','STP-SUPPORT-PLATE','Stamped Support Plate',v_supplier,v_unit,20,0,'Part number output from stamping operation',55000,'IDR',true,v_user,v_user),
    (v_wld_bracket,v_tenant,'WLD-MOUNT-BRACKET','WLD-MOUNT-BRACKET','Welded Mounting Bracket',v_supplier,v_unit,10,0,'Part number output from welding operation',185000,'IDR',true,v_user,v_user),
    (v_wld_frame,v_tenant,'WLD-SUPPORT-FRAME','WLD-SUPPORT-FRAME','Welded Support Frame',v_supplier,v_unit,10,0,'Part number output from welding operation',240000,'IDR',true,v_user,v_user)
  ON CONFLICT (tenant_id,id) DO UPDATE SET code=EXCLUDED.code,sage_item_code=EXCLUDED.sage_item_code,name=EXCLUDED.name,description=EXCLUDED.description,standard_unit_price=EXCLUDED.standard_unit_price,active=true,updated_at=now(),updated_by_user_id=EXCLUDED.updated_by_user_id;

  INSERT INTO finished_goods (id,tenant_id,item_code,name,base_unit_id,sales_price,currency,price_version,active,created_by_user_id,updated_by_user_id)
  VALUES
    (v_fg_bracket,v_tenant,'FG-CHASSIS-BRACKET','Chassis Mounting Bracket',v_unit,425000,'IDR',1,true,v_user,v_user),
    (v_fg_frame,v_tenant,'FG-CONTROL-MODULE','Control Module Frame',v_unit,575000,'IDR',1,true,v_user,v_user)
  ON CONFLICT (tenant_id,id) DO UPDATE SET item_code=EXCLUDED.item_code,name=EXCLUDED.name,sales_price=EXCLUDED.sales_price,active=true,updated_at=now(),updated_by_user_id=EXCLUDED.updated_by_user_id;

  INSERT INTO boms (id,tenant_id,output_kind,raw_material_id,finished_good_id,revision,status,notes,created_by_user_id,updated_by_user_id)
  VALUES
    (v_bom_stp_base,v_tenant,'RAW_MATERIAL',v_stp_base,NULL,1,'ACTIVE','Stamping process - base plate',v_user,v_user),
    (v_bom_stp_support,v_tenant,'RAW_MATERIAL',v_stp_support,NULL,1,'ACTIVE','Stamping process - support plate',v_user,v_user),
    (v_bom_wld_bracket,v_tenant,'RAW_MATERIAL',v_wld_bracket,NULL,1,'ACTIVE','Welding process - mounting bracket',v_user,v_user),
    (v_bom_wld_frame,v_tenant,'RAW_MATERIAL',v_wld_frame,NULL,1,'ACTIVE','Welding process - support frame',v_user,v_user),
    (v_bom_fg_bracket,v_tenant,'FG',NULL,v_fg_bracket,1,'ACTIVE','Final assembly using stamped and welded parts',v_user,v_user),
    (v_bom_fg_frame,v_tenant,'FG',NULL,v_fg_frame,1,'ACTIVE','Final assembly using stamped and welded parts',v_user,v_user)
  ON CONFLICT (tenant_id,id) DO UPDATE SET status='ACTIVE',notes=EXCLUDED.notes,updated_at=now(),updated_by_user_id=EXCLUDED.updated_by_user_id;

  INSERT INTO bom_components (tenant_id,bom_id,raw_material_id,usage_qty,sort_position)
  VALUES
    (v_tenant,v_bom_stp_base,v_sheet,1,0),(v_tenant,v_bom_stp_base,v_lube,0.05,1),
    (v_tenant,v_bom_stp_support,v_sheet,1,0),(v_tenant,v_bom_stp_support,v_lube,0.05,1),
    (v_tenant,v_bom_wld_bracket,v_stp_base,1,0),(v_tenant,v_bom_wld_bracket,v_tube,0.5,1),(v_tenant,v_bom_wld_bracket,v_wire,0.08,2),(v_tenant,v_bom_wld_bracket,v_gas,0.03,3),
    (v_tenant,v_bom_wld_frame,v_stp_support,1,0),(v_tenant,v_bom_wld_frame,v_tube,1,1),(v_tenant,v_bom_wld_frame,v_wire,0.12,2),(v_tenant,v_bom_wld_frame,v_gas,0.05,3),
    (v_tenant,v_bom_fg_bracket,v_wld_bracket,1,0),(v_tenant,v_bom_fg_bracket,v_stp_support,1,1),
    (v_tenant,v_bom_fg_frame,v_wld_frame,1,0),(v_tenant,v_bom_fg_frame,v_stp_base,2,1)
  ON CONFLICT (tenant_id,bom_id,raw_material_id) DO UPDATE SET usage_qty=EXCLUDED.usage_qty,sort_position=EXCLUDED.sort_position;
END $$;
COMMIT;
