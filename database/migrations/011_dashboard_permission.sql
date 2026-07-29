INSERT INTO permissions(code, description)
VALUES ('dashboard.view', 'View dashboard')
ON CONFLICT (code) DO UPDATE SET description = EXCLUDED.description;

INSERT INTO role_permissions(tenant_id, role_id, permission_code)
SELECT rp.tenant_id, rp.role_id, 'dashboard.view'
FROM role_permissions rp
WHERE rp.permission_code = 'inventory.view'
ON CONFLICT (tenant_id, role_id, permission_code) DO NOTHING;
