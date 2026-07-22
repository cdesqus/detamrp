BEGIN;
CREATE TABLE outgoing_sessions(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),tenant_id uuid NOT NULL REFERENCES tenants(id),document_number text NOT NULL,
 transaction_date date NOT NULL DEFAULT current_date,destination text NOT NULL CHECK(length(trim(destination)) BETWEEN 1 AND 120),notes text NOT NULL DEFAULT '',
 status text NOT NULL DEFAULT 'ACTIVE' CHECK(status IN('ACTIVE','COMPLETED','CANCELLED')),
 created_by_user_id uuid NOT NULL,created_at timestamptz NOT NULL DEFAULT now(),updated_by_user_id uuid NOT NULL,updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(tenant_id,id),UNIQUE(tenant_id,document_number),FOREIGN KEY(tenant_id,created_by_user_id) REFERENCES users(tenant_id,id),FOREIGN KEY(tenant_id,updated_by_user_id) REFERENCES users(tenant_id,id)
);
CREATE TABLE outgoing_session_scans(
 tenant_id uuid NOT NULL,session_id uuid NOT NULL,kanban_lot_id uuid NOT NULL,scanned_by_user_id uuid NOT NULL,scanned_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(tenant_id,session_id,kanban_lot_id),UNIQUE(tenant_id,kanban_lot_id),FOREIGN KEY(tenant_id,session_id) REFERENCES outgoing_sessions(tenant_id,id) ON DELETE CASCADE,FOREIGN KEY(tenant_id,kanban_lot_id) REFERENCES kanban_lots(tenant_id,id),FOREIGN KEY(tenant_id,scanned_by_user_id) REFERENCES users(tenant_id,id)
);
CREATE TABLE outgoing_documents(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),tenant_id uuid NOT NULL REFERENCES tenants(id),session_id uuid NOT NULL,document_number text NOT NULL,transaction_date date NOT NULL,destination text NOT NULL,notes text NOT NULL DEFAULT '',status text NOT NULL DEFAULT 'COMPLETED' CHECK(status='COMPLETED'),created_by_user_id uuid NOT NULL,created_at timestamptz NOT NULL DEFAULT now(),completed_by_user_id uuid NOT NULL,completed_at timestamptz NOT NULL DEFAULT now(),UNIQUE(tenant_id,id),UNIQUE(tenant_id,session_id),UNIQUE(tenant_id,document_number),FOREIGN KEY(tenant_id,session_id) REFERENCES outgoing_sessions(tenant_id,id),FOREIGN KEY(tenant_id,created_by_user_id) REFERENCES users(tenant_id,id),FOREIGN KEY(tenant_id,completed_by_user_id) REFERENCES users(tenant_id,id)
);
CREATE TABLE outgoing_kanban_lots(
 tenant_id uuid NOT NULL,outgoing_id uuid NOT NULL,kanban_lot_id uuid NOT NULL,quantity numeric(20,6) NOT NULL,PRIMARY KEY(tenant_id,outgoing_id,kanban_lot_id),UNIQUE(tenant_id,kanban_lot_id),FOREIGN KEY(tenant_id,outgoing_id) REFERENCES outgoing_documents(tenant_id,id),FOREIGN KEY(tenant_id,kanban_lot_id) REFERENCES kanban_lots(tenant_id,id)
);
ALTER TABLE outgoing_sessions ENABLE ROW LEVEL SECURITY;ALTER TABLE outgoing_sessions FORCE ROW LEVEL SECURITY;
ALTER TABLE outgoing_session_scans ENABLE ROW LEVEL SECURITY;ALTER TABLE outgoing_session_scans FORCE ROW LEVEL SECURITY;
ALTER TABLE outgoing_documents ENABLE ROW LEVEL SECURITY;ALTER TABLE outgoing_documents FORCE ROW LEVEL SECURITY;
ALTER TABLE outgoing_kanban_lots ENABLE ROW LEVEL SECURITY;ALTER TABLE outgoing_kanban_lots FORCE ROW LEVEL SECURITY;
CREATE POLICY outgoing_sessions_isolation ON outgoing_sessions USING(tenant_id=current_setting('app.tenant_id')::uuid) WITH CHECK(tenant_id=current_setting('app.tenant_id')::uuid);
CREATE POLICY outgoing_session_scans_isolation ON outgoing_session_scans USING(tenant_id=current_setting('app.tenant_id')::uuid) WITH CHECK(tenant_id=current_setting('app.tenant_id')::uuid);
CREATE POLICY outgoing_documents_isolation ON outgoing_documents USING(tenant_id=current_setting('app.tenant_id')::uuid) WITH CHECK(tenant_id=current_setting('app.tenant_id')::uuid);
CREATE POLICY outgoing_kanban_lots_isolation ON outgoing_kanban_lots USING(tenant_id=current_setting('app.tenant_id')::uuid) WITH CHECK(tenant_id=current_setting('app.tenant_id')::uuid);
GRANT SELECT,INSERT,UPDATE ON outgoing_sessions,outgoing_documents,outgoing_kanban_lots TO nextgen_app;
GRANT SELECT,INSERT,DELETE ON outgoing_session_scans TO nextgen_app;
COMMIT;
