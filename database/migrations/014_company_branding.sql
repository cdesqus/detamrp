BEGIN;

ALTER TABLE tenant_settings
  ADD COLUMN IF NOT EXISTS company_logo bytea,
  ADD COLUMN IF NOT EXISTS company_logo_mime text CHECK (company_logo_mime IN ('image/png','image/jpeg','image/webp')),
  ADD COLUMN IF NOT EXISTS login_background bytea,
  ADD COLUMN IF NOT EXISTS login_background_mime text CHECK (login_background_mime IN ('image/png','image/jpeg','image/webp'));

GRANT SELECT,UPDATE(company_logo,company_logo_mime,login_background,login_background_mime,updated_by_user_id,updated_at)
  ON tenant_settings TO nextgen_app;

CREATE OR REPLACE FUNCTION public_branding()
RETURNS TABLE(company_name text, has_logo boolean, has_login_background boolean)
LANGUAGE sql SECURITY DEFINER SET search_path = public AS $$
  SELECT COALESCE(NULLIF(BTRIM(ts.company_name),''),'DETA MRP'),
         ts.company_logo IS NOT NULL, ts.login_background IS NOT NULL
  FROM tenant_settings ts ORDER BY ts.created_at, ts.tenant_id LIMIT 1
$$;

CREATE OR REPLACE FUNCTION public_branding_media(media_kind text)
RETURNS TABLE(content bytea, content_type text)
LANGUAGE sql SECURITY DEFINER SET search_path = public AS $$
  SELECT CASE media_kind WHEN 'logo' THEN ts.company_logo WHEN 'login-background' THEN ts.login_background END,
         CASE media_kind WHEN 'logo' THEN ts.company_logo_mime WHEN 'login-background' THEN ts.login_background_mime END
  FROM tenant_settings ts
  WHERE media_kind IN ('logo','login-background')
  ORDER BY ts.created_at, ts.tenant_id LIMIT 1
$$;

REVOKE ALL ON FUNCTION public_branding() FROM PUBLIC;
REVOKE ALL ON FUNCTION public_branding_media(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public_branding(), public_branding_media(text) TO nextgen_app;

COMMIT;
