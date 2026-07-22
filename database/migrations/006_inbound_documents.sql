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
  UNIQUE (tenant_id, purchase_order_id),
  UNIQUE (tenant_id, delivery_note_number),
  FOREIGN KEY (tenant_id, purchase_order_id) REFERENCES purchase_orders(tenant_id, id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, id)
);

CREATE TABLE IF NOT EXISTS delivery_note_lines (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  delivery_note_id uuid NOT NULL,
  purchase_order_line_id uuid NOT NULL,
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_by_user_id uuid NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, delivery_note_id, purchase_order_line_id),
  UNIQUE (tenant_id, id, purchase_order_line_id),
  FOREIGN KEY (tenant_id, delivery_note_id) REFERENCES delivery_notes(tenant_id, id),
  FOREIGN KEY (tenant_id, purchase_order_line_id) REFERENCES purchase_order_lines(tenant_id, id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, id)
);

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

WITH ranked_orders AS (
  SELECT
    p.*,
    to_char(p.updated_at AT TIME ZONE 'UTC', 'YYYYMM') AS year_month,
    row_number() OVER (
      PARTITION BY p.tenant_id, to_char(p.updated_at AT TIME ZONE 'UTC', 'YYYYMM')
      ORDER BY p.updated_at, p.po_number, p.id
    ) AS ordinal
  FROM purchase_orders p
  WHERE p.status = 'APPROVED'
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
  'DN-' || p.year_month || '-' || lpad(p.ordinal::text, 5, '0'),
  'ISSUED',
  p.updated_at,
  p.updated_by_user_id,
  p.updated_at,
  p.updated_by_user_id,
  p.updated_at
FROM ranked_orders p
ON CONFLICT DO NOTHING;

INSERT INTO delivery_note_lines (
  tenant_id,
  delivery_note_id,
  purchase_order_line_id,
  created_by_user_id,
  created_at,
  updated_by_user_id,
  updated_at
)
SELECT
  p.tenant_id,
  dn.id,
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
WHERE p.status = 'APPROVED'
ON CONFLICT (tenant_id, delivery_note_id, purchase_order_line_id) DO NOTHING;

WITH ranked_lots AS (
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
    to_char(dn.issued_at AT TIME ZONE 'UTC', 'YYYYMM') AS year_month,
    row_number() OVER (
      PARTITION BY pol.tenant_id, to_char(dn.issued_at AT TIME ZONE 'UTC', 'YYYYMM')
      ORDER BY dn.issued_at, dn.delivery_note_number, pol.sort_position, pol.id, lot_no
    ) AS ordinal
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
  WHERE p.status = 'APPROVED'
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
  tenant_id,
  delivery_note_line_id,
  purchase_order_line_id,
  'KB-' || year_month || '-' || lpad(ordinal::text, 6, '0'),
  lot_number,
  quantity,
  created_by_user_id,
  created_at,
  updated_by_user_id,
  updated_at
FROM ranked_lots
ON CONFLICT (tenant_id, delivery_note_line_id, purchase_order_line_id, lot_number) DO NOTHING;

-- next_value is one greater than the largest generated ordinal, matching the
-- allocation convention used by purchase_order_number_sequences.
INSERT INTO delivery_note_number_sequences (tenant_id, year_month, next_value)
SELECT
  tenant_id,
  to_char(issued_at AT TIME ZONE 'UTC', 'YYYYMM'),
  max(right(delivery_note_number, 5)::integer) + 1
FROM delivery_notes
GROUP BY tenant_id, to_char(issued_at AT TIME ZONE 'UTC', 'YYYYMM')
ON CONFLICT (tenant_id, year_month) DO UPDATE
SET next_value = greatest(delivery_note_number_sequences.next_value, EXCLUDED.next_value);

INSERT INTO kanban_number_sequences (tenant_id, year_month, next_value)
SELECT
  tenant_id,
  substring(kanban_id FROM 4 FOR 6),
  max(right(kanban_id, 6)::integer) + 1
FROM kanban_lots
GROUP BY tenant_id, substring(kanban_id FROM 4 FOR 6)
ON CONFLICT (tenant_id, year_month) DO UPDATE
SET next_value = greatest(kanban_number_sequences.next_value, EXCLUDED.next_value);

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
