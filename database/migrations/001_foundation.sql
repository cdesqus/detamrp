CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'nextgen_app') THEN
    CREATE ROLE nextgen_app LOGIN PASSWORD 'nextgen_app' NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
  END IF;
END
$$;

CREATE TABLE tenants (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code text NOT NULL UNIQUE,
  name text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tenant_settings (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  company_name text NOT NULL,
  created_by_user_id uuid,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_by_user_id uuid,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id)
);

ALTER TABLE tenant_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_settings FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_settings_isolation ON tenant_settings
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

GRANT USAGE ON SCHEMA public TO nextgen_app;
GRANT SELECT ON tenants TO nextgen_app;
GRANT SELECT, INSERT, UPDATE ON tenant_settings TO nextgen_app;

INSERT INTO tenants (code, name)
VALUES ('OUR_COMPANY', 'Our Company')
ON CONFLICT (code) DO NOTHING;

INSERT INTO tenant_settings (tenant_id, company_name)
SELECT id, name FROM tenants WHERE code = 'OUR_COMPANY'
ON CONFLICT (tenant_id) DO NOTHING;
