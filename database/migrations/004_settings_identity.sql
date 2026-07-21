ALTER TABLE users ADD COLUMN IF NOT EXISTS email text;
UPDATE users SET email = lower(username) || '@local.invalid' WHERE email IS NULL OR btrim(email) = '';
ALTER TABLE users ALTER COLUMN email SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS users_tenant_email_ci_key ON users (tenant_id, lower(email));

ALTER TABLE roles ADD COLUMN IF NOT EXISTS active boolean NOT NULL DEFAULT true;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS created_by_user_id uuid;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS updated_by_user_id uuid;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();

ALTER TABLE tenant_settings ADD COLUMN IF NOT EXISTS default_approver_user_id uuid;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'roles_created_by_user_fk') THEN
    ALTER TABLE roles ADD CONSTRAINT roles_created_by_user_fk FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'roles_updated_by_user_fk') THEN
    ALTER TABLE roles ADD CONSTRAINT roles_updated_by_user_fk FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tenant_settings_default_approver_fk') THEN
    ALTER TABLE tenant_settings ADD CONSTRAINT tenant_settings_default_approver_fk
      FOREIGN KEY (tenant_id, default_approver_user_id) REFERENCES users(tenant_id, id);
  END IF;
END
$$;
