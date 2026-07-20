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

CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  username text NOT NULL,
  display_name text NOT NULL,
  password_hash text NOT NULL,
  locked boolean NOT NULL DEFAULT false,
  created_by_user_id uuid,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_by_user_id uuid,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, username)
);
CREATE UNIQUE INDEX users_tenant_username_ci_key ON users (tenant_id, lower(username));

CREATE TABLE sessions (
  id uuid PRIMARY KEY,
  tenant_id uuid NOT NULL,
  user_id uuid NOT NULL,
  token_hash text NOT NULL,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, token_hash),
  FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id)
);

CREATE TABLE permissions (
  code text PRIMARY KEY,
  description text NOT NULL
);

CREATE TABLE roles (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  code text NOT NULL,
  name text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, code)
);

CREATE TABLE user_roles (
  tenant_id uuid NOT NULL,
  user_id uuid NOT NULL,
  role_id uuid NOT NULL,
  PRIMARY KEY (tenant_id, user_id, role_id),
  FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, role_id) REFERENCES roles(tenant_id, id)
);

CREATE TABLE role_permissions (
  tenant_id uuid NOT NULL,
  role_id uuid NOT NULL,
  permission_code text NOT NULL REFERENCES permissions(code),
  PRIMARY KEY (tenant_id, role_id, permission_code),
  FOREIGN KEY (tenant_id, role_id) REFERENCES roles(tenant_id, id)
);

ALTER TABLE tenant_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_settings FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_settings_isolation ON tenant_settings
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
CREATE POLICY users_isolation ON users
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY sessions_isolation ON sessions
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles FORCE ROW LEVEL SECURITY;
CREATE POLICY roles_isolation ON roles USING (tenant_id = current_setting('app.tenant_id', true)::uuid) WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);
ALTER TABLE user_roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_roles FORCE ROW LEVEL SECURITY;
CREATE POLICY user_roles_isolation ON user_roles USING (tenant_id = current_setting('app.tenant_id', true)::uuid) WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);
ALTER TABLE role_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE role_permissions FORCE ROW LEVEL SECURITY;
CREATE POLICY role_permissions_isolation ON role_permissions USING (tenant_id = current_setting('app.tenant_id', true)::uuid) WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

GRANT USAGE ON SCHEMA public TO nextgen_app;
GRANT SELECT ON tenants TO nextgen_app;
GRANT SELECT, INSERT, UPDATE ON tenant_settings TO nextgen_app;
GRANT SELECT, INSERT, UPDATE ON users TO nextgen_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sessions TO nextgen_app;
GRANT SELECT ON permissions TO nextgen_app;
GRANT SELECT, INSERT, UPDATE ON roles TO nextgen_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON user_roles, role_permissions TO nextgen_app;

INSERT INTO tenants (id, code, name)
VALUES ('00000000-0000-0000-0000-000000000001', 'OUR_COMPANY', 'Our Company')
ON CONFLICT (code) DO NOTHING;

INSERT INTO tenant_settings (tenant_id, company_name)
SELECT id, name FROM tenants WHERE code = 'OUR_COMPANY'
ON CONFLICT (tenant_id) DO NOTHING;

INSERT INTO permissions (code, description) VALUES
('po.view','View supplier orders'),('po.create','Create supplier orders'),('po.edit_draft','Edit draft supplier orders'),('po.submit','Submit supplier orders'),('po.approve','Approve supplier orders'),('po.reject','Reject supplier orders'),
('po.price.view','View purchase prices'),('po.unit_price.edit','Edit purchase prices'),('dn.view','View delivery notes'),('dn.issue','Issue delivery notes'),('dn.cancel','Cancel delivery notes'),
('receiving.view','View receiving'),('receiving.create','Create receiving'),('receiving.submit','Complete receiving'),('inventory.view','View inventory'),('inventory.consume','Consume inventory'),('inventory.move','Move inventory'),
('inventory.adjust_plus','Increase inventory'),('inventory.adjust_minus','Decrease inventory'),('inventory.stock_take','Perform stock taking'),('integration.view','View integrations'),('integration.retry','Retry integrations'),
('smtp_settings.view','View SMTP settings'),('smtp_settings.manage','Manage SMTP settings'),('smtp_settings.test','Test SMTP settings'),('email_log.view','View email log'),('email_log.resend','Resend emails'),
('user.manage','Manage users'),('role.manage','Manage roles'),('configuration.manage','Manage configuration'),('master_data.view','View master data'),('master_data.manage','Manage master data')
ON CONFLICT (code) DO NOTHING;

INSERT INTO roles (tenant_id, code, name)
SELECT id, role.code, role.name FROM tenants CROSS JOIN (VALUES
('ADMIN','Administrator'),('DIRECTOR','Director'),('PURCHASING','Purchasing'),('LOGISTICS_PLANNER','Logistics Planner'),('WAREHOUSE','Warehouse'),('FINANCE','Finance'),('VIEWER','Viewer')
) AS role(code,name) WHERE tenants.code = 'OUR_COMPANY'
ON CONFLICT (tenant_id, code) DO NOTHING;

INSERT INTO role_permissions (tenant_id, role_id, permission_code)
SELECT r.tenant_id, r.id, p.code FROM roles r CROSS JOIN permissions p WHERE r.code = 'ADMIN'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (tenant_id, role_id, permission_code)
SELECT r.tenant_id, r.id, permission_code FROM roles r CROSS JOIN LATERAL unnest(
  CASE r.code
    WHEN 'DIRECTOR' THEN ARRAY['po.view','po.price.view','po.approve','po.reject','inventory.view']
    WHEN 'PURCHASING' THEN ARRAY['po.view','po.create','po.edit_draft','po.submit','po.price.view','po.unit_price.edit','master_data.view']
    WHEN 'LOGISTICS_PLANNER' THEN ARRAY['po.view','dn.view','inventory.view']
    WHEN 'WAREHOUSE' THEN ARRAY['dn.view','receiving.view','receiving.create','receiving.submit','inventory.view','inventory.consume','inventory.move','inventory.stock_take']
    WHEN 'FINANCE' THEN ARRAY['po.view','po.price.view','receiving.view','inventory.view','integration.view','email_log.view']
    WHEN 'VIEWER' THEN ARRAY['po.view','dn.view','receiving.view','inventory.view','master_data.view']
    ELSE ARRAY[]::text[]
  END
) permission_code WHERE r.code <> 'ADMIN'
ON CONFLICT DO NOTHING;
