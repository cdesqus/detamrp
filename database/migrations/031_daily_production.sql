BEGIN;
CREATE TABLE production_periods (
 tenant_id uuid NOT NULL REFERENCES tenants(id), period text NOT NULL CHECK (period ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
 closed_at timestamptz, closed_by_user_id uuid, close_reason text NOT NULL DEFAULT '',
 PRIMARY KEY(tenant_id,period), FOREIGN KEY(tenant_id,closed_by_user_id) REFERENCES users(tenant_id,id)
);
ALTER TABLE production_periods ENABLE ROW LEVEL SECURITY; ALTER TABLE production_periods FORCE ROW LEVEL SECURITY;
CREATE POLICY production_periods_isolation ON production_periods USING(tenant_id=current_setting('app.tenant_id',true)::uuid) WITH CHECK(tenant_id=current_setting('app.tenant_id',true)::uuid);
GRANT SELECT,INSERT,UPDATE ON production_periods TO nextgen_app;

ALTER TABLE production_entries ADD COLUMN entry_number text, ADD COLUMN operator_user_id uuid,
 ADD COLUMN updated_by_user_id uuid, ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now(),
 ADD COLUMN version integer NOT NULL DEFAULT 1 CHECK(version>0),
 ADD COLUMN void_reason text NOT NULL DEFAULT '', ADD COLUMN voided_by_user_id uuid,
 ADD COLUMN material_usage jsonb NOT NULL DEFAULT '[]',
 ADD COLUMN process_rate_snapshot numeric(20,6) NOT NULL DEFAULT 0 CHECK(process_rate_snapshot>=0),
 ADD COLUMN currency text NOT NULL DEFAULT 'IDR',
 ADD COLUMN effects_locked boolean NOT NULL DEFAULT false;
UPDATE production_entries SET entry_number='DP-LEGACY-'||id::text,operator_user_id=created_by_user_id,updated_by_user_id=created_by_user_id;
ALTER TABLE production_entries ALTER COLUMN entry_number SET NOT NULL, ALTER COLUMN operator_user_id SET NOT NULL, ALTER COLUMN updated_by_user_id SET NOT NULL,
 ADD UNIQUE(tenant_id,entry_number),
 ADD FOREIGN KEY(tenant_id,operator_user_id) REFERENCES users(tenant_id,id),
 ADD FOREIGN KEY(tenant_id,updated_by_user_id) REFERENCES users(tenant_id,id),
 ADD FOREIGN KEY(tenant_id,voided_by_user_id) REFERENCES users(tenant_id,id);
CREATE INDEX production_entries_operation_date_idx ON production_entries(tenant_id,operation_id,production_date);
CREATE INDEX production_entries_date_idx ON production_entries(tenant_id,production_date DESC,created_at DESC);

CREATE FUNCTION audit_daily_production() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=public AS $$
DECLARE snapshot jsonb; previous jsonb; actor_id uuid; actor_name text; event text; target uuid; code text;
BEGIN
 snapshot:=to_jsonb(NEW); previous:=CASE WHEN TG_OP='UPDATE' THEN to_jsonb(OLD) ELSE NULL END;
 actor_id:=NULLIF(current_setting('app.user_id',true),'')::uuid;
 SELECT COALESCE(NULLIF(display_name,''),username) INTO actor_name FROM users WHERE tenant_id=NEW.tenant_id AND id=actor_id;
 IF TG_TABLE_NAME='production_entries' THEN
   target:=NEW.id;code:=NEW.entry_number;
   event:=CASE WHEN TG_OP='INSERT' THEN 'CREATED' WHEN OLD.voided_at IS NULL AND NEW.voided_at IS NOT NULL THEN 'VOIDED' ELSE 'UPDATED' END;
 ELSE
   code:=NEW.period;event:='PERIOD_CLOSED';
 END IF;
 INSERT INTO activity_logs(tenant_id,actor_user_id,actor_name,module,action,target_type,target_id,target_code,before_data,after_data)
 VALUES(NEW.tenant_id,actor_id,COALESCE(actor_name,'System'),'PRODUCTION',event,TG_TABLE_NAME,target,code,previous,snapshot);
 RETURN NEW;
END; $$;
CREATE TRIGGER activity_audit AFTER INSERT OR UPDATE ON production_entries FOR EACH ROW EXECUTE FUNCTION audit_daily_production();
CREATE TRIGGER activity_audit AFTER UPDATE ON production_periods FOR EACH ROW WHEN (OLD.closed_at IS NULL AND NEW.closed_at IS NOT NULL) EXECUTE FUNCTION audit_daily_production();
CREATE FUNCTION guard_production_entry_history() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'Production entries must be voided, not deleted' USING ERRCODE='23514'; END IF;
 IF OLD.voided_at IS NOT NULL THEN RAISE EXCEPTION 'Voided entries are immutable' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END; $$;
CREATE TRIGGER production_entry_history BEFORE DELETE OR UPDATE ON production_entries FOR EACH ROW EXECUTE FUNCTION guard_production_entry_history();
REVOKE DELETE ON production_entries FROM nextgen_app;
COMMIT;
