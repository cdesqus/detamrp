BEGIN;

ALTER TABLE tenant_settings
  ADD COLUMN IF NOT EXISTS smtp_host text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS smtp_port integer NOT NULL DEFAULT 587 CHECK (smtp_port BETWEEN 1 AND 65535),
  ADD COLUMN IF NOT EXISTS smtp_security text NOT NULL DEFAULT 'STARTTLS' CHECK (smtp_security IN ('STARTTLS','TLS','NONE')),
  ADD COLUMN IF NOT EXISTS smtp_username text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS smtp_password_encrypted bytea,
  ADD COLUMN IF NOT EXISTS smtp_from_name text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS smtp_from_email text NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS email_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  email_type text NOT NULL CHECK (email_type IN ('TEST','APPROVAL','SUPPLIER')),
  reference_type text NOT NULL DEFAULT '',
  reference_id uuid,
  reference_number text NOT NULL DEFAULT '',
  recipient text NOT NULL,
  subject text NOT NULL,
  status text NOT NULL CHECK (status IN ('PENDING','SENT','FAILED')),
  attempts integer NOT NULL DEFAULT 0,
  last_error text NOT NULL DEFAULT '',
  sent_at timestamptz,
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id,id),
  FOREIGN KEY (tenant_id,created_by_user_id) REFERENCES users(tenant_id,id)
);

CREATE INDEX IF NOT EXISTS email_logs_tenant_created_idx ON email_logs(tenant_id,created_at DESC);
CREATE INDEX IF NOT EXISTS email_logs_tenant_reference_idx ON email_logs(tenant_id,reference_id,email_type);

CREATE TABLE IF NOT EXISTS approval_email_tokens (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  approval_id uuid NOT NULL,
  token_hash bytea NOT NULL,
  expires_at timestamptz NOT NULL,
  used_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id,id),
  UNIQUE (token_hash),
  FOREIGN KEY (tenant_id,approval_id) REFERENCES purchase_order_approvals(tenant_id,id)
);

ALTER TABLE email_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE email_logs FORCE ROW LEVEL SECURITY;
ALTER TABLE approval_email_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE approval_email_tokens FORCE ROW LEVEL SECURITY;

CREATE POLICY email_logs_isolation ON email_logs
  USING (tenant_id=current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id=current_setting('app.tenant_id')::uuid);
CREATE POLICY approval_email_tokens_isolation ON approval_email_tokens
  USING (tenant_id=current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id=current_setting('app.tenant_id')::uuid);

GRANT SELECT,INSERT,UPDATE ON email_logs,approval_email_tokens TO nextgen_app;
GRANT SELECT,UPDATE(smtp_host,smtp_port,smtp_security,smtp_username,smtp_password_encrypted,smtp_from_name,smtp_from_email,updated_by_user_id,updated_at) ON tenant_settings TO nextgen_app;

COMMIT;
