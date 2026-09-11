-- Rename the sample master data created by seed-live-demo.sql.
-- This keeps the existing IDs, BOMs, sales orders, and delivery history intact.
-- Run once as nextgen_app from the project directory:
--   docker compose exec -T postgres psql -U nextgen_app -d nextgen < database/rename-live-sample-data.sql
BEGIN;
SET LOCAL app.tenant_id = '00000000-0000-0000-0000-000000000001';

UPDATE suppliers
SET code='NIS-001', sage_supplier_code='NIS-001', name='Nusantara Industrial Supply',
    email='purchasing@nusantara-industrial.example', contact_person='Purchasing Desk', updated_at=now()
WHERE tenant_id='00000000-0000-0000-0000-000000000001'
  AND id='d0000000-0000-4000-8000-000000000001';

UPDATE raw_materials AS rm
SET code=names.new_code, sage_item_code=names.new_code, name=names.new_name,
    description=names.description, updated_at=now()
FROM (VALUES
  ('DEMO-STEEL','RM-STEEL-PLATE','Cold Rolled Steel Plate','Structural steel plate for assemblies'),
  ('DEMO-BOLT','RM-HEX-BOLT','Zinc Plated Hex Bolt','Standard fastening hardware'),
  ('DEMO-RUBBER','RM-RUBBER-PAD','EPDM Rubber Pad','Vibration isolation component'),
  ('DEMO-RM-4','RM-WASHER-SS','Stainless Steel Washer','Manufacturing material'),
  ('DEMO-RM-5','RM-SPACER-AL','Aluminium Spacer','Manufacturing material'),
  ('DEMO-RM-6','RM-LOCK-NUT','Nylon Lock Nut','Manufacturing material'),
  ('DEMO-RM-7','RM-CONTACT-CU','Copper Contact Strip','Manufacturing material'),
  ('DEMO-RM-8','RM-SEAL-EPDM','EPDM Seal Ring','Manufacturing material'),
  ('DEMO-RM-9','RM-GREASE','Assembly Lubricant Grease','Manufacturing material'),
  ('DEMO-RM-10','RM-CARTON','Industrial Packaging Carton','Manufacturing material'),
  ('DEMO-RM-11','RM-FOAM-INSERT','Protective Foam Insert','Manufacturing material'),
  ('DEMO-RM-12','RM-THREADLOCK','Threadlocker Adhesive','Manufacturing material')
) AS names(old_code,new_code,new_name,description)
WHERE rm.tenant_id='00000000-0000-0000-0000-000000000001'
  AND rm.code=names.old_code;

UPDATE finished_goods AS fg
SET item_code=names.new_code, name=names.new_name, updated_at=now()
FROM (VALUES
  ('DEMO-ASSEMBLY','FG-DRIVE-HOUSING','Drive Housing Assembly'),
  ('DEMO-SUBASSEMBLY','FG-SEALING-KIT','Sealing Kit Assembly'),
  ('DEMO-FG-3','FG-MOUNT-BRACKET','Pump Mounting Bracket'),
  ('DEMO-FG-4','FG-CONTROL-HOUSING','Control Panel Housing'),
  ('DEMO-FG-5','FG-COOLING-MODULE','Cooling Fan Module'),
  ('DEMO-FG-6','FG-MOTOR-DRIVE','Motor Drive Assembly'),
  ('DEMO-FG-7','FG-SENSOR-KIT','Sensor Mounting Kit'),
  ('DEMO-FG-8','FG-JUNCTION-BOX','Electrical Junction Box')
) AS names(old_code,new_code,new_name)
WHERE fg.tenant_id='00000000-0000-0000-0000-000000000001'
  AND fg.item_code=names.old_code;

UPDATE sales_order_lines AS sol
SET item_code_snapshot=fg.item_code, item_name_snapshot=fg.name
FROM finished_goods AS fg
WHERE sol.tenant_id='00000000-0000-0000-0000-000000000001'
  AND fg.tenant_id=sol.tenant_id
  AND fg.id=sol.finished_good_id
  AND (sol.item_code_snapshot LIKE 'DEMO-%' OR sol.item_name_snapshot LIKE 'Demo %');

UPDATE boms
SET notes=CASE finished_good_id
  WHEN 'd0000000-0000-4000-8000-000000000021' THEN 'Standard drive housing recipe'
  WHEN 'd0000000-0000-4000-8000-000000000022' THEN 'Standard sealing kit recipe'
  ELSE notes
END,
updated_at=now()
WHERE tenant_id='00000000-0000-0000-0000-000000000001'
  AND finished_good_id IN ('d0000000-0000-4000-8000-000000000021','d0000000-0000-4000-8000-000000000022');

COMMIT;
