package purchaseorder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationRoleResetSQL = "RESET ROLE"

func TestInboundMigrationLiveConstraintsAndChangedSourceRerun(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()

	fixture := newInboundMigrationFixture()
	if err := fixture.insertInitialSource(ctx, db); err != nil {
		t.Fatalf("insert inbound migration fixture: %v", err)
	}
	defer func() {
		if err := fixture.cleanup(ctx, db); err != nil {
			t.Errorf("cleanup inbound migration fixture: %v", err)
		}
	}()

	migration := readMigration(t, "006_inbound_documents.sql")
	applyInboundMigration(t, ctx, db, migration)

	assertInboundCounts(t, ctx, db, fixture.tenantA, 2, 2, 3)
	assertInboundCatalogAndRLS(t, ctx, db, fixture)

	var firstDNID uuid.UUID
	var firstDNNumber, secondDNNumber string
	if err := db.QueryRow(ctx, `SELECT id,delivery_note_number FROM delivery_notes WHERE tenant_id=$1 AND purchase_order_id=$2`, fixture.tenantA, fixture.po1).Scan(&firstDNID, &firstDNNumber); err != nil {
		t.Fatalf("read first delivery note: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT delivery_note_number FROM delivery_notes WHERE tenant_id=$1 AND purchase_order_id=$2`, fixture.tenantA, fixture.po2).Scan(&secondDNNumber); err != nil {
		t.Fatalf("read second delivery note: %v", err)
	}

	t.Run("rejects a DN line from another purchase order", func(t *testing.T) {
		tx := beginInboundApplicationTx(t, ctx, db, fixture.tenantA)
		defer tx.Rollback(ctx)
		_, err := tx.Exec(ctx, `INSERT INTO delivery_note_lines(tenant_id,delivery_note_id,purchase_order_id,purchase_order_line_id,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,$4,$5,$5)`, fixture.tenantA, firstDNID, fixture.po1, fixture.line2, fixture.userA)
		assertPostgresCode(t, err, "23514")
	})

	t.Run("rejects a Kanban quantity different from the PO snapshot", func(t *testing.T) {
		var deliveryNoteLineID uuid.UUID
		if err := db.QueryRow(ctx, `SELECT id FROM delivery_note_lines WHERE tenant_id=$1 AND delivery_note_id=$2 AND purchase_order_line_id=$3`, fixture.tenantA, firstDNID, fixture.line1).Scan(&deliveryNoteLineID); err != nil {
			t.Fatalf("read delivery note line: %v", err)
		}
		tx := beginInboundApplicationTx(t, ctx, db, fixture.tenantA)
		defer tx.Rollback(ctx)
		_, err := tx.Exec(ctx, `INSERT INTO kanban_lots(tenant_id,delivery_note_line_id,purchase_order_line_id,kanban_id,lot_number,quantity,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,$4,999,999,$5,$5)`, fixture.tenantA, deliveryNoteLineID, fixture.line1, "KB-202607-999999", fixture.userA)
		assertPostgresCode(t, err, "23514")
	})

	t.Run("rejects a correct-quantity Kanban beyond the source quota", func(t *testing.T) {
		var deliveryNoteLineID uuid.UUID
		if err := db.QueryRow(ctx, `SELECT id FROM delivery_note_lines WHERE tenant_id=$1 AND delivery_note_id=$2 AND purchase_order_line_id=$3`, fixture.tenantA, firstDNID, fixture.line1).Scan(&deliveryNoteLineID); err != nil {
			t.Fatalf("read delivery note line: %v", err)
		}
		tx := beginInboundApplicationTx(t, ctx, db, fixture.tenantA)
		defer tx.Rollback(ctx)
		_, err := tx.Exec(ctx, `INSERT INTO kanban_lots(tenant_id,delivery_note_line_id,purchase_order_line_id,kanban_id,lot_number,quantity,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,'KB-202607-999998',3,2.5,$4,$4)`, fixture.tenantA, deliveryNoteLineID, fixture.line1, fixture.userA)
		assertPostgresCode(t, err, "23514")
	})

	t.Run("rejects moving an existing Kanban beyond the source quota", func(t *testing.T) {
		tx := beginInboundMigrationTx(t, ctx, db)
		defer tx.Rollback(ctx)
		_, err := tx.Exec(ctx, `UPDATE kanban_lots SET lot_number=3 WHERE tenant_id=$1 AND purchase_order_line_id=$2 AND lot_number=2`, fixture.tenantA, fixture.line1)
		assertPostgresCode(t, err, "23514")
	})

	t.Run("rejects moving a DN header away from its lines", func(t *testing.T) {
		tx := beginInboundApplicationTx(t, ctx, db, fixture.tenantA)
		defer tx.Rollback(ctx)
		_, err := tx.Exec(ctx, `UPDATE delivery_notes SET purchase_order_id=$3 WHERE tenant_id=$1 AND id=$2`, fixture.tenantA, firstDNID, fixture.poOther)
		assertPostgresCode(t, err, "23503")
	})

	t.Run("rejects moving a PO line away from its DN line", func(t *testing.T) {
		tx := beginInboundApplicationTx(t, ctx, db, fixture.tenantA)
		defer tx.Rollback(ctx)
		_, err := tx.Exec(ctx, `UPDATE purchase_order_lines SET purchase_order_id=$3 WHERE tenant_id=$1 AND id=$2`, fixture.tenantA, fixture.line1, fixture.poOther)
		assertPostgresCode(t, err, "23503")
	})

	t.Run("rejects changing a PO quantity snapshot away from existing lots", func(t *testing.T) {
		tx := beginInboundApplicationTx(t, ctx, db, fixture.tenantA)
		defer tx.Rollback(ctx)
		_, err := tx.Exec(ctx, `UPDATE purchase_order_lines SET qty_per_kanban_snapshot=999 WHERE tenant_id=$1 AND id=$2`, fixture.tenantA, fixture.line1)
		assertPostgresCode(t, err, "23514")
	})

	t.Run("rejects reducing the source quota below existing lot ordinals", func(t *testing.T) {
		tx := beginInboundApplicationTx(t, ctx, db, fixture.tenantA)
		defer tx.Rollback(ctx)
		_, err := tx.Exec(ctx, `UPDATE purchase_order_lines SET total_kanban=1 WHERE tenant_id=$1 AND id=$2`, fixture.tenantA, fixture.line1)
		assertPostgresCode(t, err, "23514")
	})

	if err := fixture.insertEarlierApprovedOrder(ctx, db); err != nil {
		t.Fatalf("insert earlier approved order: %v", err)
	}
	applyInboundMigration(t, ctx, db, migration)

	t.Run("allocates collision-free numbers only for newly missing documents", func(t *testing.T) {
		assertInboundCounts(t, ctx, db, fixture.tenantA, 3, 3, 4)

		var gotFirst, gotSecond, gotThird string
		if err := db.QueryRow(ctx, `SELECT delivery_note_number FROM delivery_notes WHERE tenant_id=$1 AND purchase_order_id=$2`, fixture.tenantA, fixture.po1).Scan(&gotFirst); err != nil {
			t.Fatalf("read unchanged first delivery note: %v", err)
		}
		if err := db.QueryRow(ctx, `SELECT delivery_note_number FROM delivery_notes WHERE tenant_id=$1 AND purchase_order_id=$2`, fixture.tenantA, fixture.po2).Scan(&gotSecond); err != nil {
			t.Fatalf("read unchanged second delivery note: %v", err)
		}
		if err := db.QueryRow(ctx, `SELECT delivery_note_number FROM delivery_notes WHERE tenant_id=$1 AND purchase_order_id=$2`, fixture.tenantA, fixture.po3).Scan(&gotThird); err != nil {
			t.Fatalf("read new delivery note: %v", err)
		}
		if gotFirst != firstDNNumber || gotSecond != secondDNNumber || gotThird != "DN-202607-00003" {
			t.Fatalf("delivery note numbers after changed-source rerun = %q, %q, %q; original = %q, %q", gotFirst, gotSecond, gotThird, firstDNNumber, secondDNNumber)
		}

		var thirdKanbanID string
		if err := db.QueryRow(ctx, `SELECT k.kanban_id FROM kanban_lots k JOIN delivery_note_lines dnl ON dnl.tenant_id=k.tenant_id AND dnl.id=k.delivery_note_line_id WHERE k.tenant_id=$1 AND dnl.purchase_order_line_id=$2`, fixture.tenantA, fixture.line3).Scan(&thirdKanbanID); err != nil {
			t.Fatalf("read new Kanban ID: %v", err)
		}
		if thirdKanbanID != "KB-202607-000004" {
			t.Fatalf("new Kanban ID = %q, want KB-202607-000004", thirdKanbanID)
		}

		applyInboundMigration(t, ctx, db, migration)
		assertInboundCounts(t, ctx, db, fixture.tenantA, 3, 3, 4)
	})

	t.Run("fails capacity preflight before expanding missing lots", func(t *testing.T) {
		purchaseOrderID, lineID := uuid.New(), uuid.New()
		if _, err := db.Exec(ctx, `INSERT INTO purchase_orders(id,tenant_id,po_number,supplier_id,order_date,expected_delivery_date,currency,status,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,'PO-IN-CAPACITY',$3,'2026-07-27','2026-07-28','IDR','APPROVED',$4,$4)`, purchaseOrderID, fixture.tenantA, fixture.supplierA, fixture.userA); err != nil {
			t.Fatalf("insert capacity PO: %v", err)
		}
		if _, err := db.Exec(ctx, `INSERT INTO purchase_order_lines(id,tenant_id,purchase_order_id,raw_material_id,raw_material_code_snapshot,raw_material_name_snapshot,base_unit_id,base_unit_code_snapshot,qty_per_kanban_snapshot,total_kanban,ordered_base_qty,unit_price_snapshot,line_total,sort_position,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,$4,'IN-RM','Inbound Material',$5,'IN-KG',2.5,2,5,4.25,21.25,1,$6,$6)`, lineID, fixture.tenantA, purchaseOrderID, fixture.material, fixture.unit, fixture.userA); err != nil {
			t.Fatalf("insert capacity PO line: %v", err)
		}
		if _, err := db.Exec(ctx, `UPDATE kanban_number_sequences SET next_value=999999 WHERE tenant_id=$1 AND year_month='202607'`, fixture.tenantA); err != nil {
			t.Fatalf("exhaust Kanban sequence: %v", err)
		}
		_, err := db.Exec(ctx, migration, pgx.QueryExecModeSimpleProtocol)
		assertPostgresCode(t, err, "23514")
	})
}

type inboundMigrationFixture struct {
	tenantA, tenantB            uuid.UUID
	userA, userB                uuid.UUID
	unit, supplierA, supplierB  uuid.UUID
	material                    uuid.UUID
	po1, po2, po3, poOther, poB uuid.UUID
	line1, line2, line3         uuid.UUID
}

func newInboundMigrationFixture() inboundMigrationFixture {
	return inboundMigrationFixture{
		tenantA: uuid.New(), tenantB: uuid.New(), userA: uuid.New(), userB: uuid.New(),
		unit: uuid.New(), supplierA: uuid.New(), supplierB: uuid.New(), material: uuid.New(),
		po1: uuid.New(), po2: uuid.New(), po3: uuid.New(), poOther: uuid.New(), poB: uuid.New(),
		line1: uuid.New(), line2: uuid.New(), line3: uuid.New(),
	}
}

func (f inboundMigrationFixture) insertInitialSource(ctx context.Context, db *pgxpool.Pool) error {
	suffix := f.tenantA.String()
	statements := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO tenants(id,code,name) VALUES($1,$2,'Inbound Tenant A'),($3,$4,'Inbound Tenant B')`, []any{f.tenantA, "IN-A-" + suffix, f.tenantB, "IN-B-" + suffix}},
		{`INSERT INTO users(id,tenant_id,username,display_name,email,password_hash) VALUES($1,$2,'inbound-a','Inbound A',$3,'unused'),($4,$5,'inbound-b','Inbound B',$6,'unused')`, []any{f.userA, f.tenantA, "in-a-" + suffix + "@test.invalid", f.userB, f.tenantB, "in-b-" + suffix + "@test.invalid"}},
		{`INSERT INTO units(id,tenant_id,code,name,created_by_user_id,updated_by_user_id) VALUES($1,$2,'IN-KG','Inbound Kg',$3,$3)`, []any{f.unit, f.tenantA, f.userA}},
		{`INSERT INTO suppliers(id,tenant_id,code,sage_supplier_code,name,email,currency,created_by_user_id,updated_by_user_id) VALUES
 ($1,$2,'IN-SA',$3,'Inbound Supplier A','a@test.invalid','IDR',$4,$4),
 ($5,$6,'IN-SB',$7,'Inbound Supplier B','b@test.invalid','IDR',$8,$8)`, []any{f.supplierA, f.tenantA, "SAGE-A-" + suffix, f.userA, f.supplierB, f.tenantB, "SAGE-B-" + suffix, f.userB}},
		{`INSERT INTO raw_materials(id,tenant_id,code,sage_item_code,name,supplier_id,base_unit_id,qty_per_kanban,standard_unit_price,currency,created_by_user_id,updated_by_user_id) VALUES($1,$2,'IN-RM',$3,'Inbound Material',$4,$5,2.5,4.25,'IDR',$6,$6)`, []any{f.material, f.tenantA, "ITEM-" + suffix, f.supplierA, f.unit, f.userA}},
		{`INSERT INTO purchase_orders(id,tenant_id,po_number,supplier_id,order_date,expected_delivery_date,currency,status,created_by_user_id,created_at,updated_by_user_id,updated_at) VALUES
 ($1,$2,'PO-IN-1',$3,'2026-07-10','2026-07-11','IDR','APPROVED',$4,'2026-07-10 00:00:00+00',$4,'2026-07-10 00:00:00+00'),
 ($5,$2,'PO-IN-2',$3,'2026-07-20','2026-07-21','IDR','APPROVED',$4,'2026-07-20 00:00:00+00',$4,'2026-07-20 00:00:00+00'),
 ($6,$2,'PO-IN-OTHER',$3,'2026-07-25','2026-07-26','IDR','DRAFT',$4,'2026-07-25 00:00:00+00',$4,'2026-07-25 00:00:00+00'),
 ($7,$8,'PO-IN-B',$9,'2026-07-15','2026-07-16','IDR','APPROVED',$10,'2026-07-15 00:00:00+00',$10,'2026-07-15 00:00:00+00')`, []any{f.po1, f.tenantA, f.supplierA, f.userA, f.po2, f.poOther, f.poB, f.tenantB, f.supplierB, f.userB}},
		{`INSERT INTO purchase_order_lines(id,tenant_id,purchase_order_id,raw_material_id,raw_material_code_snapshot,raw_material_name_snapshot,base_unit_id,base_unit_code_snapshot,qty_per_kanban_snapshot,total_kanban,ordered_base_qty,unit_price_snapshot,line_total,sort_position,created_by_user_id,updated_by_user_id) VALUES
 ($1,$2,$3,$4,'IN-RM','Inbound Material',$5,'IN-KG',2.5,2,5,4.25,21.25,1,$6,$6),
 ($7,$2,$8,$4,'IN-RM','Inbound Material',$5,'IN-KG',2.5,1,2.5,4.25,10.625,1,$6,$6)`, []any{f.line1, f.tenantA, f.po1, f.material, f.unit, f.userA, f.line2, f.po2}},
	}
	for _, statement := range statements {
		if _, err := db.Exec(ctx, statement.sql, statement.args...); err != nil {
			return err
		}
	}
	return nil
}

func (f inboundMigrationFixture) insertEarlierApprovedOrder(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(ctx, `INSERT INTO purchase_orders(id,tenant_id,po_number,supplier_id,order_date,expected_delivery_date,currency,status,created_by_user_id,created_at,updated_by_user_id,updated_at)
 VALUES($1,$2,'PO-IN-0',$3,'2026-07-01','2026-07-02','IDR','APPROVED',$4,'2026-07-01 00:00:00+00',$4,'2026-07-01 00:00:00+00')`, f.po3, f.tenantA, f.supplierA, f.userA); err != nil {
		return err
	}
	_, err := db.Exec(ctx, `INSERT INTO purchase_order_lines(id,tenant_id,purchase_order_id,raw_material_id,raw_material_code_snapshot,raw_material_name_snapshot,base_unit_id,base_unit_code_snapshot,qty_per_kanban_snapshot,total_kanban,ordered_base_qty,unit_price_snapshot,line_total,sort_position,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,$4,'IN-RM','Inbound Material',$5,'IN-KG',2.5,1,2.5,4.25,10.625,1,$6,$6)`, f.line3, f.tenantA, f.po3, f.material, f.unit, f.userA)
	return err
}

func (f inboundMigrationFixture) cleanup(ctx context.Context, db *pgxpool.Pool) error {
	tables := []string{
		"kanban_lots", "delivery_note_lines", "delivery_notes", "delivery_note_number_sequences", "kanban_number_sequences",
		"purchase_order_approvals", "purchase_order_lines", "purchase_orders", "purchase_order_number_sequences",
		"raw_materials", "suppliers", "units", "users",
	}
	for _, tenantID := range []uuid.UUID{f.tenantA, f.tenantB} {
		for _, table := range tables {
			if _, err := db.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE tenant_id=$1", table), tenantID); err != nil {
				return err
			}
		}
		if _, err := db.Exec(ctx, `DELETE FROM tenants WHERE id=$1`, tenantID); err != nil {
			return err
		}
	}
	return nil
}

func applyInboundMigration(t *testing.T, ctx context.Context, db *pgxpool.Pool, migration string) {
	t.Helper()
	if _, err := db.Exec(ctx, migration, pgx.QueryExecModeSimpleProtocol); err != nil {
		t.Fatalf("apply inbound migration: %v", err)
	}
}

func assertInboundCounts(t *testing.T, ctx context.Context, db *pgxpool.Pool, tenantID uuid.UUID, wantDNs, wantLines, wantLots int) {
	t.Helper()
	var deliveryNotes, deliveryNoteLines, kanbanLots int
	if err := db.QueryRow(ctx, `SELECT
 (SELECT count(*) FROM delivery_notes WHERE tenant_id=$1),
 (SELECT count(*) FROM delivery_note_lines WHERE tenant_id=$1),
 (SELECT count(*) FROM kanban_lots WHERE tenant_id=$1)`, tenantID).Scan(&deliveryNotes, &deliveryNoteLines, &kanbanLots); err != nil {
		t.Fatalf("count inbound documents: %v", err)
	}
	if deliveryNotes != wantDNs || deliveryNoteLines != wantLines || kanbanLots != wantLots {
		t.Fatalf("inbound counts = %d/%d/%d, want %d/%d/%d", deliveryNotes, deliveryNoteLines, kanbanLots, wantDNs, wantLines, wantLots)
	}
}

func assertInboundCatalogAndRLS(t *testing.T, ctx context.Context, db *pgxpool.Pool, fixture inboundMigrationFixture) {
	t.Helper()
	tables := []string{"delivery_note_number_sequences", "kanban_number_sequences", "delivery_notes", "delivery_note_lines", "kanban_lots"}
	var securedTables, policies int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM pg_class WHERE relname=ANY($1) AND relrowsecurity AND relforcerowsecurity`, tables).Scan(&securedTables); err != nil {
		t.Fatalf("inspect inbound RLS flags: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT count(*) FROM pg_policies WHERE schemaname='public' AND tablename=ANY($1)`, tables).Scan(&policies); err != nil {
		t.Fatalf("inspect inbound policies: %v", err)
	}
	if securedTables != len(tables) || policies != len(tables) {
		t.Fatalf("inbound RLS catalog = %d secured tables/%d policies, want %d/%d", securedTables, policies, len(tables), len(tables))
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin inbound RLS check: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE nextgen_app`); err != nil {
		t.Fatalf("assume application role: %v", err)
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id',$1,true)`, fixture.tenantA.String()); err != nil {
		t.Fatalf("set inbound RLS tenant: %v", err)
	}
	var crossTenantRows int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM delivery_notes WHERE tenant_id=$1`, fixture.tenantB).Scan(&crossTenantRows); err != nil {
		t.Fatalf("query inbound documents through RLS: %v", err)
	}
	if crossTenantRows != 0 {
		t.Fatalf("application role saw %d cross-tenant delivery notes", crossTenantRows)
	}
}

func assertPostgresCode(t *testing.T, err error, want string) {
	t.Helper()
	var databaseError *pgconn.PgError
	if !errors.As(err, &databaseError) || databaseError.Code != want {
		t.Fatalf("database error = %T %v, want PostgreSQL code %s", err, err, want)
	}
}

func beginInboundMigrationTx(t *testing.T, ctx context.Context, db *pgxpool.Pool) pgx.Tx {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin inbound migration transaction: %v", err)
	}
	if _, err := tx.Exec(ctx, migrationRoleResetSQL); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("reset inbound migration role: %v", err)
	}
	return tx
}

func beginInboundApplicationTx(t *testing.T, ctx context.Context, db *pgxpool.Pool, tenantID uuid.UUID) pgx.Tx {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin inbound application transaction: %v", err)
	}
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE nextgen_app`); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("assume application role: %v", err)
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id',$1,true)`, tenantID.String()); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("set inbound application tenant: %v", err)
	}
	return tx
}
