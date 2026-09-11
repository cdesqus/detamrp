-- Refresh frozen calculation snapshots for sample Sales Orders after master-data renames.
-- Historical business orders are left untouched; only the sample finished-good IDs
-- and legacy DEMO-* snapshots are selected.
-- Run after rename-live-sample-data.sql:
--   docker compose exec -T postgres psql -U nextgen_app -d nextgen < database/refresh-live-sample-snapshots.sql
BEGIN;
SET LOCAL app.tenant_id = '00000000-0000-0000-0000-000000000001';

CREATE OR REPLACE FUNCTION pg_temp.refresh_sample_snapshot_node(node jsonb)
RETURNS jsonb
LANGUAGE plpgsql
AS $$
DECLARE
  item_code text;
  item_name text;
  child jsonb;
  children jsonb := '[]'::jsonb;
BEGIN
  IF node->>'kind' = 'FG' THEN
    SELECT item_code, name INTO item_code, item_name
    FROM finished_goods
    WHERE tenant_id=current_setting('app.tenant_id')::uuid
      AND id=(node->>'itemId')::uuid;
  ELSE
    SELECT code, name INTO item_code, item_name
    FROM raw_materials
    WHERE tenant_id=current_setting('app.tenant_id')::uuid
      AND id=(node->>'itemId')::uuid;
  END IF;

  IF item_code IS NOT NULL THEN
    node := jsonb_set(node, '{itemCode}', to_jsonb(item_code), true);
    node := jsonb_set(node, '{name}', to_jsonb(item_name), true);
  END IF;

  IF jsonb_typeof(node->'children') = 'array' THEN
    FOR child IN SELECT value FROM jsonb_array_elements(node->'children') LOOP
      children := children || jsonb_build_array(pg_temp.refresh_sample_snapshot_node(child));
    END LOOP;
    node := jsonb_set(node, '{children}', children, true);
  END IF;
  RETURN node;
END;
$$;

UPDATE sales_order_lines AS line
SET item_code_snapshot=fg.item_code,
    item_name_snapshot=fg.name,
    calculation_snapshot=CASE
      WHEN line.calculation_snapshot IS NULL THEN line.calculation_snapshot
      ELSE jsonb_set(line.calculation_snapshot, '{root}', pg_temp.refresh_sample_snapshot_node(line.calculation_snapshot->'root'), true)
    END
FROM finished_goods AS fg
WHERE line.tenant_id='00000000-0000-0000-0000-000000000001'
  AND fg.tenant_id=line.tenant_id
  AND fg.id=line.finished_good_id
  AND (
    line.item_code_snapshot LIKE 'DEMO-%'
    OR line.item_name_snapshot LIKE 'Demo %'
    OR fg.item_code IN ('FG-DRIVE-HOUSING','FG-SEALING-KIT','FG-CHASSIS-BRACKET','FG-CONTROL-MODULE')
  );

COMMIT;
