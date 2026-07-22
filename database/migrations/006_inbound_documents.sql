CREATE TABLE IF NOT EXISTS delivery_note_number_sequences (
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  year_month char(6) NOT NULL CHECK (year_month ~ '^[0-9]{6}$'),
  next_value integer NOT NULL DEFAULT 1 CHECK (next_value > 0),
  PRIMARY KEY (tenant_id, year_month)
);

CREATE TABLE IF NOT EXISTS kanban_number_sequences (
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  year_month char(6) NOT NULL CHECK (year_month ~ '^[0-9]{6}$'),
  next_value integer NOT NULL DEFAULT 1 CHECK (next_value > 0),
  PRIMARY KEY (tenant_id, year_month)
);

CREATE TABLE IF NOT EXISTS delivery_notes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  purchase_order_id uuid NOT NULL,
  delivery_note_number text NOT NULL CHECK (delivery_note_number ~ '^DN-[0-9]{6}-[0-9]{5}$'),
  status text NOT NULL DEFAULT 'ISSUED' CHECK (status IN ('ISSUED','PARTIALLY_RECEIVED','RECEIVED','CANCELLED')),
  issued_at timestamptz NOT NULL DEFAULT now(),
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_by_user_id uuid NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  CONSTRAINT delivery_notes_tenant_id_purchase_order_key UNIQUE (tenant_id, id, purchase_order_id),
  UNIQUE (tenant_id, purchase_order_id),
  UNIQUE (tenant_id, delivery_note_number),
  FOREIGN KEY (tenant_id, purchase_order_id) REFERENCES purchase_orders(tenant_id, id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, id)
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'purchase_order_lines_tenant_order_line_key') THEN
    ALTER TABLE purchase_order_lines ADD CONSTRAINT purchase_order_lines_tenant_order_line_key
      UNIQUE (tenant_id, purchase_order_id, id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'delivery_notes_tenant_id_purchase_order_key') THEN
    ALTER TABLE delivery_notes ADD CONSTRAINT delivery_notes_tenant_id_purchase_order_key
      UNIQUE (tenant_id, id, purchase_order_id);
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS delivery_note_lines (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  delivery_note_id uuid NOT NULL,
  purchase_order_id uuid NOT NULL,
  purchase_order_line_id uuid NOT NULL,
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_by_user_id uuid NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, delivery_note_id, purchase_order_line_id),
  UNIQUE (tenant_id, id, purchase_order_line_id),
  CONSTRAINT delivery_note_lines_header_order_fk
    FOREIGN KEY (tenant_id, delivery_note_id, purchase_order_id)
    REFERENCES delivery_notes(tenant_id, id, purchase_order_id),
  CONSTRAINT delivery_note_lines_order_line_fk
    FOREIGN KEY (tenant_id, purchase_order_id, purchase_order_line_id)
    REFERENCES purchase_order_lines(tenant_id, purchase_order_id, id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, id)
);

ALTER TABLE delivery_note_lines ADD COLUMN IF NOT EXISTS purchase_order_id uuid;
UPDATE delivery_note_lines dnl
SET purchase_order_id = pol.purchase_order_id
FROM purchase_order_lines pol
WHERE pol.tenant_id = dnl.tenant_id
  AND pol.id = dnl.purchase_order_line_id
  AND dnl.purchase_order_id IS NULL;
ALTER TABLE delivery_note_lines ALTER COLUMN purchase_order_id SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'delivery_note_lines_header_order_fk') THEN
    ALTER TABLE delivery_note_lines ADD CONSTRAINT delivery_note_lines_header_order_fk
      FOREIGN KEY (tenant_id, delivery_note_id, purchase_order_id)
      REFERENCES delivery_notes(tenant_id, id, purchase_order_id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'delivery_note_lines_order_line_fk') THEN
    ALTER TABLE delivery_note_lines ADD CONSTRAINT delivery_note_lines_order_line_fk
      FOREIGN KEY (tenant_id, purchase_order_id, purchase_order_line_id)
      REFERENCES purchase_order_lines(tenant_id, purchase_order_id, id);
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS kanban_lots (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  delivery_note_line_id uuid NOT NULL,
  purchase_order_line_id uuid NOT NULL,
  kanban_id text NOT NULL CHECK (kanban_id ~ '^KB-[0-9]{6}-[0-9]{6}$'),
  lot_number integer NOT NULL CHECK (lot_number > 0),
  quantity numeric(20,6) NOT NULL CHECK (quantity > 0),
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_by_user_id uuid NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, kanban_id),
  UNIQUE (tenant_id, delivery_note_line_id, purchase_order_line_id, lot_number),
  FOREIGN KEY (tenant_id, delivery_note_line_id, purchase_order_line_id)
    REFERENCES delivery_note_lines(tenant_id, id, purchase_order_line_id),
  FOREIGN KEY (tenant_id, purchase_order_line_id) REFERENCES purchase_order_lines(tenant_id, id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, id)
);

CREATE OR REPLACE FUNCTION delivery_note_line_matches_purchase_order()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  delivery_note_purchase_order_id uuid;
  line_purchase_order_id uuid;
BEGIN
  SELECT purchase_order_id INTO delivery_note_purchase_order_id
  FROM delivery_notes
  WHERE tenant_id = NEW.tenant_id AND id = NEW.delivery_note_id;

  SELECT purchase_order_id INTO line_purchase_order_id
  FROM purchase_order_lines
  WHERE tenant_id = NEW.tenant_id AND id = NEW.purchase_order_line_id;

  IF delivery_note_purchase_order_id IS NULL OR
     line_purchase_order_id IS NULL OR
     delivery_note_purchase_order_id <> NEW.purchase_order_id OR
     line_purchase_order_id <> NEW.purchase_order_id OR
     delivery_note_purchase_order_id <> line_purchase_order_id THEN
    RAISE EXCEPTION 'delivery note line must belong to the delivery note purchase order'
      USING ERRCODE = '23514';
  END IF;
  RETURN NEW;
END;
$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'delivery_note_line_purchase_order_check' AND tgrelid = 'delivery_note_lines'::regclass) THEN
    EXECUTE 'CREATE TRIGGER delivery_note_line_purchase_order_check
      BEFORE INSERT OR UPDATE OF tenant_id, delivery_note_id, purchase_order_id, purchase_order_line_id ON delivery_note_lines
      FOR EACH ROW EXECUTE FUNCTION delivery_note_line_matches_purchase_order()';
  END IF;
END
$$;

CREATE OR REPLACE FUNCTION purchase_order_line_preserves_kanban_quantity()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM kanban_lots
    WHERE tenant_id = OLD.tenant_id
      AND purchase_order_line_id = OLD.id
      AND quantity <> NEW.qty_per_kanban_snapshot
  ) THEN
    RAISE EXCEPTION 'purchase order line quantity snapshot must equal existing Kanban lot quantities'
      USING ERRCODE = '23514';
  END IF;
  IF EXISTS (
    SELECT 1
    FROM kanban_lots
    WHERE tenant_id = OLD.tenant_id
      AND purchase_order_line_id = OLD.id
      AND lot_number > NEW.total_kanban
  ) THEN
    RAISE EXCEPTION 'purchase order line Kanban quota cannot be lower than existing lot ordinals'
      USING ERRCODE = '23514';
  END IF;
  RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS purchase_order_line_kanban_quantity_check ON purchase_order_lines;
CREATE TRIGGER purchase_order_line_kanban_quantity_check
  BEFORE UPDATE OF qty_per_kanban_snapshot, total_kanban ON purchase_order_lines
  FOR EACH ROW EXECUTE FUNCTION purchase_order_line_preserves_kanban_quantity();

CREATE OR REPLACE FUNCTION kanban_lot_quantity_matches_purchase_order_line()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  delivery_note_line_purchase_order_line_id uuid;
  expected_quantity numeric(20,6);
  expected_total_kanban numeric(20,6);
BEGIN
  SELECT purchase_order_line_id INTO delivery_note_line_purchase_order_line_id
  FROM delivery_note_lines
  WHERE tenant_id = NEW.tenant_id AND id = NEW.delivery_note_line_id;

  SELECT qty_per_kanban_snapshot, total_kanban INTO expected_quantity, expected_total_kanban
  FROM purchase_order_lines
  WHERE tenant_id = NEW.tenant_id AND id = NEW.purchase_order_line_id;

  IF delivery_note_line_purchase_order_line_id IS NULL OR
     expected_quantity IS NULL OR
     expected_total_kanban IS NULL OR
     delivery_note_line_purchase_order_line_id <> NEW.purchase_order_line_id OR
     expected_quantity <> NEW.quantity OR
     NEW.lot_number > expected_total_kanban THEN
    RAISE EXCEPTION 'Kanban lot quantity must equal the purchase order line quantity snapshot'
      USING ERRCODE = '23514';
  END IF;
  RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS kanban_lot_quantity_check ON kanban_lots;
CREATE TRIGGER kanban_lot_quantity_check
  BEFORE INSERT OR UPDATE OF tenant_id, delivery_note_line_id, purchase_order_line_id, lot_number, quantity ON kanban_lots
  FOR EACH ROW EXECUTE FUNCTION kanban_lot_quantity_matches_purchase_order_line();

-- Reconcile sequence state before allocating missing backfill rows. Allocation
-- then advances next_value as one block per tenant/month, so reruns never
-- renumber existing documents when an earlier source order appears later.
INSERT INTO delivery_note_number_sequences (tenant_id, year_month, next_value)
SELECT
  tenant_id,
  to_char(issued_at AT TIME ZONE 'UTC', 'YYYYMM'),
  max(right(delivery_note_number, 5)::integer) + 1
FROM delivery_notes
GROUP BY tenant_id, to_char(issued_at AT TIME ZONE 'UTC', 'YYYYMM')
ON CONFLICT (tenant_id, year_month) DO UPDATE
SET next_value = greatest(delivery_note_number_sequences.next_value, EXCLUDED.next_value);

WITH missing_orders AS (
  SELECT
    p.*,
    to_char(p.updated_at AT TIME ZONE 'UTC', 'YYYYMM') AS year_month
  FROM purchase_orders p
  LEFT JOIN delivery_notes dn
    ON dn.tenant_id = p.tenant_id AND dn.purchase_order_id = p.id
  WHERE p.status = 'APPROVED' AND dn.id IS NULL
), ranked_missing_orders AS (
  SELECT
    p.*,
    row_number() OVER (
      PARTITION BY p.tenant_id, p.year_month
      ORDER BY p.updated_at, p.po_number, p.id
    )::integer AS missing_ordinal
  FROM missing_orders p
), missing_order_counts AS (
  SELECT tenant_id, year_month, count(*)::integer AS missing_count
  FROM ranked_missing_orders
  GROUP BY tenant_id, year_month
), allocated_delivery_note_ranges AS (
  INSERT INTO delivery_note_number_sequences (tenant_id, year_month, next_value)
  SELECT tenant_id, year_month, missing_count + 1
  FROM missing_order_counts
  ON CONFLICT (tenant_id, year_month) DO UPDATE
  SET next_value = delivery_note_number_sequences.next_value + EXCLUDED.next_value - 1
  RETURNING tenant_id, year_month, next_value
)
INSERT INTO delivery_notes (
  tenant_id,
  purchase_order_id,
  delivery_note_number,
  status,
  issued_at,
  created_by_user_id,
  created_at,
  updated_by_user_id,
  updated_at
)
SELECT
  p.tenant_id,
  p.id,
  'DN-' || p.year_month || '-' || lpad((r.next_value - c.missing_count + p.missing_ordinal - 1)::text, 5, '0'),
  'ISSUED',
  p.updated_at,
  p.updated_by_user_id,
  p.updated_at,
  p.updated_by_user_id,
  p.updated_at
FROM ranked_missing_orders p
JOIN missing_order_counts c
  ON c.tenant_id = p.tenant_id AND c.year_month = p.year_month
JOIN allocated_delivery_note_ranges r
  ON r.tenant_id = p.tenant_id AND r.year_month = p.year_month
ON CONFLICT (tenant_id, purchase_order_id) DO NOTHING;

INSERT INTO delivery_note_lines (
  tenant_id,
  delivery_note_id,
  purchase_order_id,
  purchase_order_line_id,
  created_by_user_id,
  created_at,
  updated_by_user_id,
  updated_at
)
SELECT
  p.tenant_id,
  dn.id,
  p.id,
  pol.id,
  p.updated_by_user_id,
  p.updated_at,
  p.updated_by_user_id,
  p.updated_at
FROM purchase_orders p
JOIN delivery_notes dn
  ON dn.tenant_id = p.tenant_id AND dn.purchase_order_id = p.id
JOIN purchase_order_lines pol
  ON pol.tenant_id = p.tenant_id AND pol.purchase_order_id = p.id
LEFT JOIN delivery_note_lines dnl
  ON dnl.tenant_id = p.tenant_id
 AND dnl.delivery_note_id = dn.id
 AND dnl.purchase_order_line_id = pol.id
WHERE p.status = 'APPROVED' AND dnl.id IS NULL
ON CONFLICT (tenant_id, delivery_note_id, purchase_order_line_id) DO NOTHING;

INSERT INTO kanban_number_sequences (tenant_id, year_month, next_value)
SELECT
  tenant_id,
  substring(kanban_id FROM 4 FOR 6),
  max(right(kanban_id, 6)::integer) + 1
FROM kanban_lots
GROUP BY tenant_id, substring(kanban_id FROM 4 FOR 6)
ON CONFLICT (tenant_id, year_month) DO UPDATE
SET next_value = greatest(kanban_number_sequences.next_value, EXCLUDED.next_value);

DO $$
BEGIN
  IF EXISTS (
    WITH line_capacity AS (
      SELECT
        pol.tenant_id,
        to_char(dn.issued_at AT TIME ZONE 'UTC', 'YYYYMM') AS year_month,
        greatest(pol.total_kanban::bigint - count(kl.id), 0) AS missing_count
      FROM purchase_orders p
      JOIN delivery_notes dn
        ON dn.tenant_id = p.tenant_id AND dn.purchase_order_id = p.id
      JOIN purchase_order_lines pol
        ON pol.tenant_id = p.tenant_id AND pol.purchase_order_id = p.id
      JOIN delivery_note_lines dnl
        ON dnl.tenant_id = pol.tenant_id
       AND dnl.delivery_note_id = dn.id
       AND dnl.purchase_order_line_id = pol.id
      LEFT JOIN kanban_lots kl
        ON kl.tenant_id = dnl.tenant_id
       AND kl.delivery_note_line_id = dnl.id
       AND kl.purchase_order_line_id = pol.id
      WHERE p.status = 'APPROVED'
      GROUP BY pol.tenant_id, dn.issued_at, pol.id, pol.total_kanban
    ), monthly_capacity AS (
      SELECT tenant_id, year_month, sum(missing_count)::bigint AS missing_count
      FROM line_capacity
      GROUP BY tenant_id, year_month
    )
    SELECT 1
    FROM monthly_capacity c
    LEFT JOIN kanban_number_sequences s
      ON s.tenant_id = c.tenant_id AND s.year_month = c.year_month
    WHERE coalesce(s.next_value, 1)::bigint + c.missing_count - 1 > 999999
  ) THEN
    RAISE EXCEPTION 'inbound document capacity preflight failed: six-digit Kanban range exhausted'
      USING ERRCODE = '23514';
  END IF;
END
$$;

WITH expected_lots AS (
  SELECT
    pol.tenant_id,
    dnl.id AS delivery_note_line_id,
    pol.id AS purchase_order_line_id,
    pol.qty_per_kanban_snapshot AS quantity,
    lot_no::integer AS lot_number,
    dn.created_by_user_id,
    dn.created_at,
    dn.updated_by_user_id,
    dn.updated_at,
    to_char(dn.issued_at AT TIME ZONE 'UTC', 'YYYYMM') AS year_month
  FROM purchase_orders p
  JOIN delivery_notes dn
    ON dn.tenant_id = p.tenant_id AND dn.purchase_order_id = p.id
  JOIN purchase_order_lines pol
    ON pol.tenant_id = p.tenant_id AND pol.purchase_order_id = p.id
  JOIN delivery_note_lines dnl
    ON dnl.tenant_id = pol.tenant_id
   AND dnl.delivery_note_id = dn.id
   AND dnl.purchase_order_line_id = pol.id
  CROSS JOIN LATERAL generate_series(1, pol.total_kanban::bigint) lot_no
  LEFT JOIN kanban_lots existing_lot
    ON existing_lot.tenant_id = pol.tenant_id
   AND existing_lot.delivery_note_line_id = dnl.id
   AND existing_lot.purchase_order_line_id = pol.id
   AND existing_lot.lot_number = lot_no
  WHERE p.status = 'APPROVED' AND existing_lot.id IS NULL
), ranked_missing_lots AS (
  SELECT
    expected_lots.*,
    row_number() OVER (
      PARTITION BY tenant_id, year_month
      ORDER BY created_at, delivery_note_line_id, purchase_order_line_id, lot_number
    )::integer AS missing_ordinal
  FROM expected_lots
), missing_lot_counts AS (
  SELECT tenant_id, year_month, count(*)::integer AS missing_count
  FROM ranked_missing_lots
  GROUP BY tenant_id, year_month
), allocated_kanban_ranges AS (
  INSERT INTO kanban_number_sequences (tenant_id, year_month, next_value)
  SELECT tenant_id, year_month, missing_count + 1
  FROM missing_lot_counts
  ON CONFLICT (tenant_id, year_month) DO UPDATE
  SET next_value = kanban_number_sequences.next_value + EXCLUDED.next_value - 1
  RETURNING tenant_id, year_month, next_value
)
INSERT INTO kanban_lots (
  tenant_id,
  delivery_note_line_id,
  purchase_order_line_id,
  kanban_id,
  lot_number,
  quantity,
  created_by_user_id,
  created_at,
  updated_by_user_id,
  updated_at
)
SELECT
  k.tenant_id,
  k.delivery_note_line_id,
  k.purchase_order_line_id,
  'KB-' || k.year_month || '-' || lpad((r.next_value - c.missing_count + k.missing_ordinal - 1)::text, 6, '0'),
  k.lot_number,
  k.quantity,
  k.created_by_user_id,
  k.created_at,
  k.updated_by_user_id,
  k.updated_at
FROM ranked_missing_lots k
JOIN missing_lot_counts c
  ON c.tenant_id = k.tenant_id AND c.year_month = k.year_month
JOIN allocated_kanban_ranges r
  ON r.tenant_id = k.tenant_id AND r.year_month = k.year_month
ON CONFLICT (tenant_id, delivery_note_line_id, purchase_order_line_id, lot_number) DO NOTHING;

CREATE INDEX IF NOT EXISTS delivery_notes_tenant_status_issued_at_idx
  ON delivery_notes (tenant_id, status, issued_at DESC);
CREATE INDEX IF NOT EXISTS delivery_note_lines_tenant_delivery_note_idx
  ON delivery_note_lines (tenant_id, delivery_note_id);
CREATE INDEX IF NOT EXISTS kanban_lots_tenant_delivery_note_line_idx
  ON kanban_lots (tenant_id, delivery_note_line_id, lot_number);

ALTER TABLE delivery_note_number_sequences ENABLE ROW LEVEL SECURITY;
ALTER TABLE delivery_note_number_sequences FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'delivery_note_number_sequences' AND policyname = 'delivery_note_number_sequences_isolation') THEN
    EXECUTE 'CREATE POLICY delivery_note_number_sequences_isolation ON delivery_note_number_sequences
      USING (tenant_id = current_setting(''app.tenant_id'')::uuid)
      WITH CHECK (tenant_id = current_setting(''app.tenant_id'')::uuid)';
  END IF;
END $$;

ALTER TABLE kanban_number_sequences ENABLE ROW LEVEL SECURITY;
ALTER TABLE kanban_number_sequences FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'kanban_number_sequences' AND policyname = 'kanban_number_sequences_isolation') THEN
    EXECUTE 'CREATE POLICY kanban_number_sequences_isolation ON kanban_number_sequences
      USING (tenant_id = current_setting(''app.tenant_id'')::uuid)
      WITH CHECK (tenant_id = current_setting(''app.tenant_id'')::uuid)';
  END IF;
END $$;

ALTER TABLE delivery_notes ENABLE ROW LEVEL SECURITY;
ALTER TABLE delivery_notes FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'delivery_notes' AND policyname = 'delivery_notes_isolation') THEN
    EXECUTE 'CREATE POLICY delivery_notes_isolation ON delivery_notes
      USING (tenant_id = current_setting(''app.tenant_id'')::uuid)
      WITH CHECK (tenant_id = current_setting(''app.tenant_id'')::uuid)';
  END IF;
END $$;

ALTER TABLE delivery_note_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE delivery_note_lines FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'delivery_note_lines' AND policyname = 'delivery_note_lines_isolation') THEN
    EXECUTE 'CREATE POLICY delivery_note_lines_isolation ON delivery_note_lines
      USING (tenant_id = current_setting(''app.tenant_id'')::uuid)
      WITH CHECK (tenant_id = current_setting(''app.tenant_id'')::uuid)';
  END IF;
END $$;

ALTER TABLE kanban_lots ENABLE ROW LEVEL SECURITY;
ALTER TABLE kanban_lots FORCE ROW LEVEL SECURITY;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = 'kanban_lots' AND policyname = 'kanban_lots_isolation') THEN
    EXECUTE 'CREATE POLICY kanban_lots_isolation ON kanban_lots
      USING (tenant_id = current_setting(''app.tenant_id'')::uuid)
      WITH CHECK (tenant_id = current_setting(''app.tenant_id'')::uuid)';
  END IF;
END $$;

GRANT SELECT, INSERT, UPDATE ON delivery_note_number_sequences, kanban_number_sequences TO nextgen_app;
GRANT SELECT, INSERT, UPDATE ON delivery_notes TO nextgen_app;
GRANT SELECT, INSERT ON delivery_note_lines, kanban_lots TO nextgen_app;
