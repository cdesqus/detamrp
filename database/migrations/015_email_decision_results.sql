BEGIN;

ALTER TABLE email_logs
  DROP CONSTRAINT IF EXISTS email_logs_email_type_check;

ALTER TABLE email_logs
  ADD CONSTRAINT email_logs_email_type_check
  CHECK (email_type IN ('TEST','APPROVAL','SUPPLIER','DECISION'));

COMMIT;
