BEGIN;
-- Production reports in Rupiah. Any other currency needs a conversion rate
-- before it may be released, so no report ever mixes denominations.
ALTER TABLE tenant_settings ADD COLUMN currency_rates jsonb NOT NULL DEFAULT '{}'::jsonb
  CHECK (jsonb_typeof(currency_rates) = 'object');
COMMENT ON COLUMN tenant_settings.currency_rates IS 'Currency code to IDR rate, for example {"USD": 16250}. IDR is always 1.';
COMMIT;
