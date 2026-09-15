BEGIN;
-- Production consumes material by quantity, not by whole kanban lots, so the
-- inventory ledger gains lot-less production events. Stock on hand becomes the
-- kanban balance less what production has issued.
ALTER TABLE inventory_ledger_entries ALTER COLUMN kanban_lot_id DROP NOT NULL;
ALTER TABLE inventory_ledger_entries DROP CONSTRAINT inventory_ledger_entries_event_type_check;
ALTER TABLE inventory_ledger_entries ADD CONSTRAINT inventory_ledger_entries_event_type_check
  CHECK (event_type IN ('RECEIVING','OUTGOING','PRODUCTION_ISSUE','PRODUCTION_RETURN'));
-- Receiving and outgoing still move whole lots; production never does.
ALTER TABLE inventory_ledger_entries ADD CONSTRAINT inventory_ledger_production_has_no_lot
  CHECK ((event_type IN ('PRODUCTION_ISSUE','PRODUCTION_RETURN')) = (kanban_lot_id IS NULL));
ALTER TABLE inventory_ledger_entries ADD CONSTRAINT inventory_ledger_production_direction
  CHECK (event_type<>'PRODUCTION_ISSUE' OR quantity_delta<0);
ALTER TABLE inventory_ledger_entries ADD CONSTRAINT inventory_ledger_return_direction
  CHECK (event_type<>'PRODUCTION_RETURN' OR quantity_delta>0);

CREATE INDEX inventory_ledger_material_idx ON inventory_ledger_entries(tenant_id,raw_material_id,event_type);
CREATE INDEX inventory_ledger_reference_idx ON inventory_ledger_entries(tenant_id,reference_type,reference_id);
COMMIT;
