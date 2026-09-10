-- Safely disable the demo data created by seed-live-demo.sql.
-- Existing transactions remain readable for audit; master rows are not hard-deleted.
BEGIN;
SET LOCAL app.tenant_id = '00000000-0000-0000-0000-000000000001';
UPDATE boms SET status='REVISED', updated_at=now()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND id IN ('d0000000-0000-4000-8000-000000000031','d0000000-0000-4000-8000-000000000032');
UPDATE finished_goods SET active=false, updated_at=now()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND id IN ('d0000000-0000-4000-8000-000000000021','d0000000-0000-4000-8000-000000000022');
UPDATE raw_materials SET active=false, updated_at=now()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND id IN ('d0000000-0000-4000-8000-000000000011','d0000000-0000-4000-8000-000000000012','d0000000-0000-4000-8000-000000000013');
UPDATE suppliers SET active=false, updated_at=now()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND id = 'd0000000-0000-4000-8000-000000000001';
COMMIT;
