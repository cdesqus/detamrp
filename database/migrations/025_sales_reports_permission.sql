BEGIN;
INSERT INTO permissions(code,description) VALUES('sales_report.view','View sales, material requirement, and customer delivery reports') ON CONFLICT(code) DO UPDATE SET description=EXCLUDED.description;
INSERT INTO role_permissions(tenant_id,role_id,permission_code) SELECT r.tenant_id,r.id,'sales_report.view' FROM roles r WHERE r.code='ADMIN' ON CONFLICT DO NOTHING;
COMMIT;
