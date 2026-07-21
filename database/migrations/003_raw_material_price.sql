ALTER TABLE raw_materials ADD COLUMN IF NOT EXISTS standard_unit_price numeric(20,6) NOT NULL DEFAULT 0;
ALTER TABLE raw_materials ADD COLUMN IF NOT EXISTS currency char(3);

UPDATE raw_materials rm
SET currency = s.currency
FROM suppliers s
WHERE s.tenant_id = rm.tenant_id AND s.id = rm.supplier_id AND rm.currency IS NULL;

ALTER TABLE raw_materials ALTER COLUMN currency SET NOT NULL;

DO $$ BEGIN
  ALTER TABLE raw_materials ADD CONSTRAINT raw_materials_standard_unit_price_nonnegative CHECK (standard_unit_price >= 0);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
