BEGIN;

INSERT INTO permissions(code, description) VALUES
  ('bom.view', 'View BOMs'),
  ('bom.manage', 'Manage BOM drafts'),
  ('bom.activate', 'Activate BOMs')
ON CONFLICT(code) DO UPDATE SET description = EXCLUDED.description;

INSERT INTO role_permissions(tenant_id, role_id, permission_code)
SELECT r.tenant_id, r.id, p.code
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'ADMIN'
  AND p.code IN ('bom.view', 'bom.manage', 'bom.activate')
ON CONFLICT DO NOTHING;

COMMIT;
