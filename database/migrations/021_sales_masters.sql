BEGIN;

CREATE TABLE customers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  code text NOT NULL,
  name text NOT NULL,
  address text NOT NULL DEFAULT '',
  contact text NOT NULL DEFAULT '',
  email text NOT NULL DEFAULT '',
  phone text NOT NULL DEFAULT '',
  active boolean NOT NULL DEFAULT true,
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_by_user_id uuid NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, code),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, id)
);

CREATE TABLE finished_goods (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  item_code text NOT NULL,
  name text NOT NULL,
  base_unit_id uuid NOT NULL,
  sales_price numeric(20,6) NOT NULL CHECK (sales_price > 0),
  currency text NOT NULL CHECK (currency IN ('IDR', 'USD', 'EUR', 'JPY', 'SGD')),
  price_version integer NOT NULL DEFAULT 1 CHECK (price_version > 0),
  active boolean NOT NULL DEFAULT true,
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_by_user_id uuid NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, item_code),
  FOREIGN KEY (tenant_id, base_unit_id) REFERENCES units(tenant_id, id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, id)
);

ALTER TABLE customers ENABLE ROW LEVEL SECURITY;
ALTER TABLE customers FORCE ROW LEVEL SECURITY;
CREATE POLICY customers_isolation ON customers
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

ALTER TABLE finished_goods ENABLE ROW LEVEL SECURITY;
ALTER TABLE finished_goods FORCE ROW LEVEL SECURITY;
CREATE POLICY finished_goods_isolation ON finished_goods
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

REVOKE ALL ON customers, finished_goods FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON customers, finished_goods TO nextgen_app;

INSERT INTO permissions (code, description) VALUES
  ('customer.view', 'View customers'),
  ('customer.manage', 'Manage customers'),
  ('fg.view', 'View finished goods'),
  ('fg.manage', 'Manage finished goods'),
  ('fg.price.manage', 'Manage finished-good prices')
ON CONFLICT (code) DO UPDATE SET description = EXCLUDED.description;

INSERT INTO role_permissions (tenant_id, role_id, permission_code)
SELECT r.tenant_id, r.id, p.code
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'ADMIN'
  AND p.code IN ('customer.view', 'customer.manage', 'fg.view', 'fg.manage', 'fg.price.manage')
ON CONFLICT DO NOTHING;

CREATE TRIGGER activity_audit
AFTER INSERT OR UPDATE OR DELETE ON customers
FOR EACH ROW EXECUTE FUNCTION record_activity_change();

CREATE TRIGGER activity_audit
AFTER INSERT OR UPDATE OR DELETE ON finished_goods
FOR EACH ROW EXECUTE FUNCTION record_activity_change();

COMMIT;
