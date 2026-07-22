BEGIN;

ALTER TABLE purchase_orders DROP CONSTRAINT IF EXISTS purchase_orders_status_check;
ALTER TABLE purchase_orders ADD CONSTRAINT purchase_orders_status_check
  CHECK (status IN ('DRAFT','PENDING_APPROVAL','APPROVED','PARTIALLY_RECEIVED','FULLY_RECEIVED','REJECTED','CANCELLED'));

ALTER TABLE kanban_lots ADD COLUMN IF NOT EXISTS status text NOT NULL DEFAULT 'ISSUED';
ALTER TABLE kanban_lots DROP CONSTRAINT IF EXISTS kanban_lots_status_check;
ALTER TABLE kanban_lots ADD CONSTRAINT kanban_lots_status_check CHECK (status IN ('ISSUED','IN_STOCK','CONSUMED'));

CREATE TABLE receiving_sessions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id),
 delivery_note_id uuid NOT NULL, receiving_number text NOT NULL,
 status text NOT NULL DEFAULT 'ACTIVE' CHECK(status IN ('ACTIVE','PAUSED','COMPLETED','CANCELLED','EXPIRED')),
 receiving_date date NOT NULL DEFAULT current_date, cancel_reason text NOT NULL DEFAULT '',
 created_by_user_id uuid NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 updated_by_user_id uuid NOT NULL, updated_at timestamptz NOT NULL DEFAULT now(), expires_at timestamptz NOT NULL DEFAULT now()+interval '30 minutes',
 UNIQUE(tenant_id,id), UNIQUE(tenant_id,receiving_number),
 FOREIGN KEY(tenant_id,delivery_note_id) REFERENCES delivery_notes(tenant_id,id),
 FOREIGN KEY(tenant_id,created_by_user_id) REFERENCES users(tenant_id,id),
 FOREIGN KEY(tenant_id,updated_by_user_id) REFERENCES users(tenant_id,id)
);
CREATE UNIQUE INDEX one_open_receiving_session_per_dn ON receiving_sessions(tenant_id,delivery_note_id)
 WHERE status IN ('ACTIVE','PAUSED');

CREATE TABLE receiving_session_scans (
 tenant_id uuid NOT NULL, session_id uuid NOT NULL, kanban_lot_id uuid NOT NULL,
 scanned_by_user_id uuid NOT NULL, scanned_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(tenant_id,session_id,kanban_lot_id), UNIQUE(tenant_id,kanban_lot_id),
 FOREIGN KEY(tenant_id,session_id) REFERENCES receiving_sessions(tenant_id,id) ON DELETE CASCADE,
 FOREIGN KEY(tenant_id,kanban_lot_id) REFERENCES kanban_lots(tenant_id,id),
 FOREIGN KEY(tenant_id,scanned_by_user_id) REFERENCES users(tenant_id,id)
);

CREATE TABLE receivings (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), session_id uuid NOT NULL,
 delivery_note_id uuid NOT NULL, purchase_order_id uuid NOT NULL, receiving_number text NOT NULL,
 receiving_date date NOT NULL, status text NOT NULL DEFAULT 'COMPLETED' CHECK(status='COMPLETED'),
 sage_receipt_number text NOT NULL DEFAULT '', planned_kanban integer NOT NULL,
 previously_received_kanban integer NOT NULL, received_now_kanban integer NOT NULL, outstanding_kanban integer NOT NULL,
 created_by_user_id uuid NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), completed_by_user_id uuid NOT NULL, completed_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(tenant_id,id), UNIQUE(tenant_id,session_id), UNIQUE(tenant_id,receiving_number),
 FOREIGN KEY(tenant_id,session_id) REFERENCES receiving_sessions(tenant_id,id),
 FOREIGN KEY(tenant_id,delivery_note_id) REFERENCES delivery_notes(tenant_id,id),
 FOREIGN KEY(tenant_id,purchase_order_id) REFERENCES purchase_orders(tenant_id,id),
 FOREIGN KEY(tenant_id,created_by_user_id) REFERENCES users(tenant_id,id),
 FOREIGN KEY(tenant_id,completed_by_user_id) REFERENCES users(tenant_id,id)
);

CREATE TABLE receiving_kanban_lots (
 tenant_id uuid NOT NULL, receiving_id uuid NOT NULL, kanban_lot_id uuid NOT NULL,
 quantity numeric(20,6) NOT NULL, PRIMARY KEY(tenant_id,receiving_id,kanban_lot_id), UNIQUE(tenant_id,kanban_lot_id),
 FOREIGN KEY(tenant_id,receiving_id) REFERENCES receivings(tenant_id,id),
 FOREIGN KEY(tenant_id,kanban_lot_id) REFERENCES kanban_lots(tenant_id,id)
);

CREATE TABLE inventory_ledger_entries (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id),
 event_type text NOT NULL CHECK(event_type IN ('RECEIVING','OUTGOING')), kanban_lot_id uuid NOT NULL,
 raw_material_id uuid NOT NULL, quantity_delta numeric(20,6) NOT NULL CHECK(quantity_delta<>0),
 base_unit_code text NOT NULL, warehouse text NOT NULL DEFAULT 'RAW MATERIAL', location text NOT NULL DEFAULT 'DEFAULT',
 reference_type text NOT NULL, reference_id uuid NOT NULL, created_by_user_id uuid NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(tenant_id,id), UNIQUE(tenant_id,event_type,kanban_lot_id),
 FOREIGN KEY(tenant_id,kanban_lot_id) REFERENCES kanban_lots(tenant_id,id),
 FOREIGN KEY(tenant_id,raw_material_id) REFERENCES raw_materials(tenant_id,id),
 FOREIGN KEY(tenant_id,created_by_user_id) REFERENCES users(tenant_id,id)
);

CREATE TABLE integration_outbox (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id),
 event_type text NOT NULL, aggregate_id uuid NOT NULL, idempotency_key text NOT NULL,
 payload jsonb NOT NULL, status text NOT NULL DEFAULT 'PENDING' CHECK(status IN ('PENDING','RETRYING','SYNCED','FAILED')),
 attempts integer NOT NULL DEFAULT 0, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(tenant_id,id), UNIQUE(tenant_id,idempotency_key)
);

CREATE OR REPLACE FUNCTION reject_inventory_ledger_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'inventory ledger is append-only' USING ERRCODE='23514'; END $$;
CREATE TRIGGER inventory_ledger_append_only BEFORE UPDATE OR DELETE ON inventory_ledger_entries
 FOR EACH ROW EXECUTE FUNCTION reject_inventory_ledger_mutation();

ALTER TABLE receiving_sessions ENABLE ROW LEVEL SECURITY; ALTER TABLE receiving_sessions FORCE ROW LEVEL SECURITY;
ALTER TABLE receiving_session_scans ENABLE ROW LEVEL SECURITY; ALTER TABLE receiving_session_scans FORCE ROW LEVEL SECURITY;
ALTER TABLE receivings ENABLE ROW LEVEL SECURITY; ALTER TABLE receivings FORCE ROW LEVEL SECURITY;
ALTER TABLE receiving_kanban_lots ENABLE ROW LEVEL SECURITY; ALTER TABLE receiving_kanban_lots FORCE ROW LEVEL SECURITY;
ALTER TABLE inventory_ledger_entries ENABLE ROW LEVEL SECURITY; ALTER TABLE inventory_ledger_entries FORCE ROW LEVEL SECURITY;
ALTER TABLE integration_outbox ENABLE ROW LEVEL SECURITY; ALTER TABLE integration_outbox FORCE ROW LEVEL SECURITY;

CREATE POLICY receiving_sessions_isolation ON receiving_sessions USING(tenant_id=current_setting('app.tenant_id')::uuid) WITH CHECK(tenant_id=current_setting('app.tenant_id')::uuid);
CREATE POLICY receiving_session_scans_isolation ON receiving_session_scans USING(tenant_id=current_setting('app.tenant_id')::uuid) WITH CHECK(tenant_id=current_setting('app.tenant_id')::uuid);
CREATE POLICY receivings_isolation ON receivings USING(tenant_id=current_setting('app.tenant_id')::uuid) WITH CHECK(tenant_id=current_setting('app.tenant_id')::uuid);
CREATE POLICY receiving_kanban_lots_isolation ON receiving_kanban_lots USING(tenant_id=current_setting('app.tenant_id')::uuid) WITH CHECK(tenant_id=current_setting('app.tenant_id')::uuid);
CREATE POLICY inventory_ledger_entries_isolation ON inventory_ledger_entries USING(tenant_id=current_setting('app.tenant_id')::uuid) WITH CHECK(tenant_id=current_setting('app.tenant_id')::uuid);
CREATE POLICY integration_outbox_isolation ON integration_outbox USING(tenant_id=current_setting('app.tenant_id')::uuid) WITH CHECK(tenant_id=current_setting('app.tenant_id')::uuid);

GRANT SELECT,INSERT,UPDATE ON receiving_sessions,receivings,receiving_kanban_lots,integration_outbox TO nextgen_app;
GRANT SELECT,INSERT,DELETE ON receiving_session_scans TO nextgen_app;
GRANT SELECT,INSERT ON inventory_ledger_entries TO nextgen_app;
GRANT UPDATE(status,updated_by_user_id,updated_at) ON kanban_lots TO nextgen_app;
GRANT UPDATE(status,updated_by_user_id,updated_at) ON purchase_orders TO nextgen_app;
GRANT UPDATE(status,updated_by_user_id,updated_at) ON delivery_notes TO nextgen_app;

COMMIT;
