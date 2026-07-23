BEGIN;

ALTER TABLE outgoing_sessions
  DROP CONSTRAINT IF EXISTS outgoing_sessions_destination_check;

ALTER TABLE outgoing_sessions
  ADD CONSTRAINT outgoing_sessions_destination_check
  CHECK (length(trim(destination)) <= 120);

COMMIT;
