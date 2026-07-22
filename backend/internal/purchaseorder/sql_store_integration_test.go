package purchaseorder

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/database"
)

func TestSQLStoreLivePurchaseOrderWorkflow(t *testing.T) {
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatalf("open admin pool: %v", err)
	}
	fixture := newLiveFixture()
	if err := fixture.insert(ctx, admin); err != nil {
		admin.Close()
		t.Fatalf("insert fixture: %v", err)
	}
	defer func() {
		if err := fixture.cleanup(ctx, admin); err != nil {
			t.Errorf("cleanup fixture: %v", err)
		}
		admin.Close()
	}()

	appDB, err := database.Open(ctx, applicationDatabaseURL(t, adminURL))
	if err != nil {
		t.Fatalf("open application pool: %v", err)
	}
	defer appDB.Close()
	store := NewSQLStore(appDB)
	buyer := Actor{TenantID: fixture.tenantA, UserID: fixture.buyer, DisplayName: "Live Buyer"}
	input := OrderInput{
		SupplierID:           fixture.supplier,
		OrderDate:            time.Date(2026, time.July, 21, 0, 0, 0, 0, time.UTC),
		ExpectedDeliveryDate: time.Date(2026, time.July, 22, 0, 0, 0, 0, time.UTC),
		Currency:             "IDR",
		Lines:                []LineInput{{RawMaterialID: fixture.material, TotalKanban: decimal.NewFromInt(3)}},
	}

	order, err := store.CreateOrder(ctx, buyer, input)
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.SupplierName != "Live Supplier" || len(order.Lines) != 1 {
		t.Fatalf("unexpected order snapshots: %#v", order)
	}
	line := order.Lines[0]
	if line.RawMaterialCode != "LIVE-RM" || line.RawMaterialName != "Live Material" || line.BaseUnitCode != "KG" ||
		!line.QtyPerKanbanSnapshot.Equal(decimal.RequireFromString("2.500000")) ||
		!line.UnitPriceSnapshot.Equal(decimal.RequireFromString("4.250000")) ||
		!line.LineTotal.Equal(decimal.RequireFromString("31.875000")) {
		t.Fatalf("incorrect line snapshot: %#v", line)
	}

	otherTenant := Actor{TenantID: fixture.tenantB, UserID: fixture.otherUser}
	if _, err := store.GetOrder(ctx, otherTenant, order.ID); err == nil {
		t.Fatal("other tenant read purchase order")
	} else {
		var missing NotFoundError
		if !errors.As(err, &missing) {
			t.Fatalf("other tenant read error = %T %v, want not found", err, err)
		}
	}
	assertRLSRejectsCrossTenantWrite(t, ctx, applicationDatabaseURL(t, adminURL), fixture)

	submitted, err := store.SubmitOrder(ctx, buyer, order.ID)
	if err != nil {
		t.Fatalf("submit order: %v", err)
	}
	if submitted.Status != StatusPendingApproval || submitted.SubmittedApproverUserID != fixture.approver ||
		submitted.SubmittedApproverDisplayName != "Live Director" || submitted.SubmittedApproverEmail != "director@live.test" {
		t.Fatalf("incorrect submitted approver snapshot: %#v", submitted)
	}
	approver := Actor{TenantID: fixture.tenantA, UserID: fixture.approver, DisplayName: "Live Director"}
	approvals, total, err := store.ListApprovals(ctx, approver, ListQuery{Limit: 50})
	if err != nil || total != 1 || len(approvals) != 1 {
		t.Fatalf("list approvals = %d/%d, err %v", len(approvals), total, err)
	}
	if _, err := store.Approve(ctx, approver, approvals[0].ID, DecisionInput{}); err != nil {
		t.Fatalf("approve order: %v", err)
	}
	if _, err := store.Reject(ctx, approver, approvals[0].ID, DecisionInput{Reason: "late"}); err == nil {
		t.Fatal("second approval decision unexpectedly succeeded")
	} else {
		var conflict ConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("second decision error = %T %v, want conflict", err, err)
		}
	}

	results := make(chan Order, 2)
	errorsFound := make(chan error, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			created, createErr := store.CreateOrder(ctx, buyer, input)
			if createErr != nil {
				errorsFound <- createErr
				return
			}
			results <- created
		}()
	}
	group.Wait()
	close(results)
	close(errorsFound)
	for createErr := range errorsFound {
		t.Errorf("concurrent create: %v", createErr)
	}
	numbers := map[string]struct{}{}
	for created := range results {
		numbers[created.PONumber] = struct{}{}
	}
	if len(numbers) != 2 {
		t.Fatalf("concurrent PO numbers were not unique: %#v", numbers)
	}
}

func TestSQLStoreApprovalGeneratesDocuments(t *testing.T) {
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatalf("open admin pool: %v", err)
	}
	fixture := newLiveFixture()
	if err := fixture.insert(ctx, admin); err != nil {
		admin.Close()
		t.Fatalf("insert fixture: %v", err)
	}
	defer func() {
		if err := fixture.cleanup(ctx, admin); err != nil {
			t.Errorf("cleanup fixture: %v", err)
		}
		admin.Close()
	}()

	appDB, err := database.Open(ctx, applicationDatabaseURL(t, adminURL))
	if err != nil {
		t.Fatalf("open application pool: %v", err)
	}
	defer appDB.Close()
	store := NewSQLStore(appDB)
	buyer := Actor{TenantID: fixture.tenantA, UserID: fixture.buyer, DisplayName: "Live Buyer"}
	approver := Actor{TenantID: fixture.tenantA, UserID: fixture.approver, DisplayName: "Live Director"}
	input := OrderInput{
		SupplierID:           fixture.supplier,
		OrderDate:            time.Date(2026, time.July, 21, 0, 0, 0, 0, time.UTC),
		ExpectedDeliveryDate: time.Date(2026, time.July, 22, 0, 0, 0, 0, time.UTC),
		Currency:             "IDR",
		Lines: []LineInput{
			{RawMaterialID: fixture.material, TotalKanban: decimal.NewFromInt(2)},
			{RawMaterialID: fixture.material2, TotalKanban: decimal.NewFromInt(3)},
		},
	}

	order, err := store.CreateOrder(ctx, buyer, input)
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if _, err := store.SubmitOrder(ctx, buyer, order.ID); err != nil {
		t.Fatalf("submit order: %v", err)
	}
	approvals, total, err := store.ListApprovals(ctx, approver, ListQuery{Limit: 50})
	if err != nil || total != 1 || len(approvals) != 1 {
		t.Fatalf("list approvals = %d/%d, err %v", len(approvals), total, err)
	}
	if _, err := store.Approve(ctx, approver, approvals[0].ID, DecisionInput{}); err != nil {
		t.Fatalf("approve order: %v", err)
	}
	assertApprovedDocumentCounts(t, ctx, admin, fixture.tenantA, order.ID, 1, 2, 5)
	assertHydratedDocumentSummary(t, ctx, store, buyer, order.ID, true)

	if _, err := admin.Exec(ctx, `UPDATE purchase_orders SET status='DRAFT' WHERE tenant_id=$1 AND id=$2`, fixture.tenantA, order.ID); err != nil {
		t.Fatalf("create stale non-approved document fixture: %v", err)
	}
	assertHydratedDocumentSummary(t, ctx, store, buyer, order.ID, false)
	if _, err := admin.Exec(ctx, `UPDATE purchase_orders SET status='APPROVED' WHERE tenant_id=$1 AND id=$2`, fixture.tenantA, order.ID); err != nil {
		t.Fatalf("restore approved document fixture: %v", err)
	}

	if err := database.WithTenant(ctx, appDB, tenantContext(approver), func(tx database.TenantTx) error {
		if err := tx.QueryRow(ctx, `SELECT id FROM purchase_orders WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, fixture.tenantA, order.ID).Scan(&order.ID); err != nil {
			return err
		}
		return ensureApprovedDocuments(ctx, tx, approver, order.ID)
	}); err != nil {
		t.Fatalf("retry approved document generation: %v", err)
	}
	assertApprovedDocumentCounts(t, ctx, admin, fixture.tenantA, order.ID, 1, 2, 5)

	failedOrder, err := store.CreateOrder(ctx, buyer, OrderInput{
		SupplierID:           fixture.supplier,
		OrderDate:            time.Date(2026, time.July, 21, 0, 0, 0, 0, time.UTC),
		ExpectedDeliveryDate: time.Date(2026, time.July, 22, 0, 0, 0, 0, time.UTC),
		Currency:             "IDR",
		Lines:                []LineInput{{RawMaterialID: fixture.material, TotalKanban: decimal.NewFromInt(2)}},
	})
	if err != nil {
		t.Fatalf("create forced-error order: %v", err)
	}
	if _, err := store.SubmitOrder(ctx, buyer, failedOrder.ID); err != nil {
		t.Fatalf("submit forced-error order: %v", err)
	}
	failedApprovals, total, err := store.ListApprovals(ctx, approver, ListQuery{Limit: 50})
	if err != nil || total != 1 || len(failedApprovals) != 1 {
		t.Fatalf("list forced-error approvals = %d/%d, err %v", len(failedApprovals), total, err)
	}
	forcedSequence, err := admin.Exec(ctx, `UPDATE kanban_number_sequences SET next_value=999999 WHERE tenant_id=$1`, fixture.tenantA)
	if err != nil {
		t.Fatalf("force generator error: %v", err)
	}
	if forcedSequence.RowsAffected() != 1 {
		t.Fatalf("forced generator sequences = %d, want 1", forcedSequence.RowsAffected())
	}
	if _, err := store.Approve(ctx, approver, failedApprovals[0].ID, DecisionInput{}); err == nil {
		t.Fatal("approval with forced generator error unexpectedly succeeded")
	} else {
		var capacity CapacityError
		if !errors.As(err, &capacity) || capacity.Field != "documents" {
			t.Fatalf("forced generator error = %T %v, want typed document capacity error", err, err)
		}
	}

	var approvalStatus ApprovalStatus
	var orderStatus Status
	var failedDocumentCount int
	if err := admin.QueryRow(ctx, `SELECT a.status,p.status,
	 (SELECT count(*) FROM delivery_notes dn WHERE dn.tenant_id=p.tenant_id AND dn.purchase_order_id=p.id)
	 FROM purchase_order_approvals a
	 JOIN purchase_orders p ON p.tenant_id=a.tenant_id AND p.id=a.purchase_order_id
	 WHERE a.tenant_id=$1 AND a.id=$2`, fixture.tenantA, failedApprovals[0].ID).Scan(&approvalStatus, &orderStatus, &failedDocumentCount); err != nil {
		t.Fatalf("read forced-error statuses: %v", err)
	}
	if approvalStatus != ApprovalPending || orderStatus != StatusPendingApproval || failedDocumentCount != 0 {
		t.Fatalf("forced-error state = approval %s, order %s, documents %d", approvalStatus, orderStatus, failedDocumentCount)
	}
}

func TestSQLStoreConcurrentApprovalsAllocateUniqueDocumentsAndRejectDuplicateDecision(t *testing.T) {
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatalf("open admin pool: %v", err)
	}
	fixture := newLiveFixture()
	if err := fixture.insert(ctx, admin); err != nil {
		admin.Close()
		t.Fatalf("insert fixture: %v", err)
	}
	defer func() {
		if err := fixture.cleanup(ctx, admin); err != nil {
			t.Errorf("cleanup fixture: %v", err)
		}
		admin.Close()
	}()

	appDB, err := database.Open(ctx, applicationDatabaseURL(t, adminURL))
	if err != nil {
		t.Fatalf("open application pool: %v", err)
	}
	defer appDB.Close()
	store := NewSQLStore(appDB)
	buyer := Actor{TenantID: fixture.tenantA, UserID: fixture.buyer, DisplayName: "Live Buyer"}
	approver := Actor{TenantID: fixture.tenantA, UserID: fixture.approver, DisplayName: "Live Director"}
	var orders []Order
	for index, count := range []int64{2, 3} {
		order, err := store.CreateOrder(ctx, buyer, OrderInput{
			SupplierID: fixture.supplier, OrderDate: time.Date(2026, time.July, 21+index, 0, 0, 0, 0, time.UTC),
			ExpectedDeliveryDate: time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC), Currency: "IDR",
			Lines: []LineInput{{RawMaterialID: fixture.material, TotalKanban: decimal.NewFromInt(count)}},
		})
		if err != nil {
			t.Fatalf("create concurrent order %d: %v", index, err)
		}
		if _, err := store.SubmitOrder(ctx, buyer, order.ID); err != nil {
			t.Fatalf("submit concurrent order %d: %v", index, err)
		}
		orders = append(orders, order)
	}
	approvals, total, err := store.ListApprovals(ctx, approver, ListQuery{Limit: 50})
	if err != nil || total != 2 || len(approvals) != 2 {
		t.Fatalf("list concurrent approvals = %d/%d, err %v", len(approvals), total, err)
	}

	errorsByApproval := make(chan error, len(approvals))
	var wait sync.WaitGroup
	for _, approval := range approvals {
		wait.Add(1)
		go func(approvalID uuid.UUID) {
			defer wait.Done()
			_, approveErr := store.Approve(ctx, approver, approvalID, DecisionInput{})
			errorsByApproval <- approveErr
		}(approval.ID)
	}
	wait.Wait()
	close(errorsByApproval)
	for approveErr := range errorsByApproval {
		if approveErr != nil {
			t.Fatalf("concurrent approval: %v", approveErr)
		}
	}
	assertApprovedDocumentCounts(t, ctx, admin, fixture.tenantA, orders[0].ID, 1, 1, 2)
	assertApprovedDocumentCounts(t, ctx, admin, fixture.tenantA, orders[1].ID, 1, 1, 3)
	var distinctDNs, distinctKanbans int
	if err := admin.QueryRow(ctx, `SELECT count(DISTINCT delivery_note_number),
 (SELECT count(DISTINCT kanban_id) FROM kanban_lots WHERE tenant_id=$1)
 FROM delivery_notes WHERE tenant_id=$1`, fixture.tenantA).Scan(&distinctDNs, &distinctKanbans); err != nil {
		t.Fatalf("count concurrent identifiers: %v", err)
	}
	if distinctDNs != 2 || distinctKanbans != 5 {
		t.Fatalf("concurrent identifiers = %d DNs/%d Kanbans, want 2/5", distinctDNs, distinctKanbans)
	}
	if _, err := store.Approve(ctx, approver, approvals[0].ID, DecisionInput{}); err == nil {
		t.Fatal("duplicate approval decision unexpectedly succeeded")
	} else {
		var conflict ConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("duplicate approval error = %T %v, want conflict", err, err)
		}
	}
	assertApprovedDocumentCounts(t, ctx, admin, fixture.tenantA, orders[0].ID, 1, 1, 2)
	assertApprovedDocumentCounts(t, ctx, admin, fixture.tenantA, orders[1].ID, 1, 1, 3)
}

func assertHydratedDocumentSummary(t *testing.T, ctx context.Context, store *SQLStore, actor Actor, orderID uuid.UUID, wantDocuments bool) {
	t.Helper()
	detail, err := store.GetOrder(ctx, actor, orderID)
	if err != nil {
		t.Fatalf("get order document summary: %v", err)
	}
	items, total, err := store.ListOrders(ctx, actor, ListQuery{Limit: 50})
	if err != nil {
		t.Fatalf("list order document summaries: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != orderID {
		t.Fatalf("list order document summaries = %d/%d %#v", len(items), total, items)
	}
	for _, result := range []Order{detail, items[0]} {
		if !wantDocuments {
			if result.Documents != nil {
				t.Fatalf("non-approved documents = %#v, want nil", result.Documents)
			}
			continue
		}
		if result.Documents == nil || result.Documents.DeliveryNoteID == uuid.Nil || result.Documents.DeliveryNoteNumber == "" || result.Documents.KanbanCount != 5 || result.Documents.IssuedAt.IsZero() {
			t.Fatalf("approved documents = %#v, want generated summary", result.Documents)
		}
	}
}

func TestValidateApprovedDocumentLinesRejectsUnrepresentableKanbanTotal(t *testing.T) {
	lineID := uuid.New()
	err := validateApprovedDocumentLines(uuid.New(), []approvedDocumentLine{{
		ID:          lineID,
		TotalKanban: decimal.NewFromInt(1_000_000),
	}})
	if err == nil || !strings.Contains(err.Error(), "exceeds monthly identifier capacity") {
		t.Fatalf("validate million-Kanban line = %v, want identifier-capacity error", err)
	}
}

func assertApprovedDocumentCounts(t *testing.T, ctx context.Context, db *pgxpool.Pool, tenantID, purchaseOrderID uuid.UUID, wantDNs, wantLines, wantLots int) {
	t.Helper()
	var dnCount, dnLineCount, lotCount int
	if err := db.QueryRow(ctx, `SELECT
	 (SELECT count(*) FROM delivery_notes WHERE tenant_id=$1 AND purchase_order_id=$2),
	 (SELECT count(*) FROM delivery_note_lines WHERE tenant_id=$1 AND purchase_order_id=$2),
	 (SELECT count(*) FROM kanban_lots k
	  JOIN purchase_order_lines pol ON pol.tenant_id=k.tenant_id AND pol.id=k.purchase_order_line_id
	  WHERE k.tenant_id=$1 AND pol.purchase_order_id=$2)`, tenantID, purchaseOrderID).Scan(&dnCount, &dnLineCount, &lotCount); err != nil {
		t.Fatalf("count approved documents: %v", err)
	}
	if dnCount != wantDNs || dnLineCount != wantLines || lotCount != wantLots {
		t.Fatalf("documents = dn %d, lines %d, lots %d", dnCount, dnLineCount, lotCount)
	}
}

type liveFixture struct {
	tenantA, tenantB, buyer, approver, otherUser     uuid.UUID
	role, measurement, supplier, material, material2 uuid.UUID
}

func newLiveFixture() liveFixture {
	return liveFixture{
		tenantA: uuid.New(), tenantB: uuid.New(), buyer: uuid.New(), approver: uuid.New(), otherUser: uuid.New(),
		role: uuid.New(), measurement: uuid.New(), supplier: uuid.New(), material: uuid.New(), material2: uuid.New(),
	}
}

func (f liveFixture) insert(ctx context.Context, db *pgxpool.Pool) error {
	statements := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO tenants(id,code,name) VALUES($1,$2,'Live Tenant A'),($3,$4,'Live Tenant B')`, []any{f.tenantA, "LIVE-" + f.tenantA.String(), f.tenantB, "LIVE-" + f.tenantB.String()}},
		{`INSERT INTO users(id,tenant_id,username,display_name,email,password_hash) VALUES
 ($1,$2,'buyer','Live Buyer','buyer@live.test','unused'),($3,$2,'director','Live Director','director@live.test','unused'),
 ($4,$5,'other','Other Tenant User','other@live.test','unused')`, []any{f.buyer, f.tenantA, f.approver, f.otherUser, f.tenantB}},
		{`INSERT INTO tenant_settings(tenant_id,company_name,default_approver_user_id) VALUES($1,'Live Tenant A',$2),($3,'Live Tenant B',NULL)`, []any{f.tenantA, f.approver, f.tenantB}},
		{`INSERT INTO roles(id,tenant_id,code,name,active) VALUES($1,$2,'LIVE_DIRECTOR','Live Director',true)`, []any{f.role, f.tenantA}},
		{`INSERT INTO role_permissions(tenant_id,role_id,permission_code) VALUES($1,$2,'po.approve')`, []any{f.tenantA, f.role}},
		{`INSERT INTO user_roles(tenant_id,user_id,role_id) VALUES($1,$2,$3)`, []any{f.tenantA, f.approver, f.role}},
		{`INSERT INTO measurements(id,tenant_id,code,name,created_by_user_id,updated_by_user_id) VALUES($1,$2,'KG','Kilogram',$3,$3)`, []any{f.measurement, f.tenantA, f.buyer}},
		{`INSERT INTO suppliers(id,tenant_id,code,sage_supplier_code,name,email,currency,created_by_user_id,updated_by_user_id) VALUES($1,$2,'LIVE-SUP','LIVE-SAGE','Live Supplier','supplier@live.test','IDR',$3,$3)`, []any{f.supplier, f.tenantA, f.buyer}},
		{`INSERT INTO raw_materials(id,tenant_id,code,sage_item_code,name,supplier_id,base_unit_id,qty_per_kanban,standard_unit_price,currency,created_by_user_id,updated_by_user_id) VALUES
 ($1,$2,'LIVE-RM','LIVE-ITEM','Live Material',$3,$4,2.5,4.25,'IDR',$5,$5),
 ($6,$2,'LIVE-RM-2','LIVE-ITEM-2','Live Material 2',$3,$4,4,6.5,'IDR',$5,$5)`, []any{f.material, f.tenantA, f.supplier, f.measurement, f.buyer, f.material2}},
	}
	for _, statement := range statements {
		if _, err := db.Exec(ctx, statement.sql, statement.args...); err != nil {
			return err
		}
	}
	return nil
}

func (f liveFixture) cleanup(ctx context.Context, db *pgxpool.Pool) error {
	tables := []string{"kanban_lots", "delivery_note_lines", "delivery_notes", "delivery_note_number_sequences", "kanban_number_sequences", "purchase_order_approvals", "purchase_order_lines", "purchase_orders", "purchase_order_number_sequences", "role_permissions", "user_roles", "raw_materials", "suppliers", "measurements", "roles", "tenant_settings", "sessions", "users"}
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

func applicationDatabaseURL(t *testing.T, adminURL string) string {
	t.Helper()
	parsed, err := url.Parse(adminURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	parsed.User = url.UserPassword("nextgen_app", "nextgen_app")
	return parsed.String()
}

func assertRLSRejectsCrossTenantWrite(t *testing.T, ctx context.Context, databaseURL string, fixture liveFixture) {
	t.Helper()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open RLS pool: %v", err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin RLS transaction: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id',$1,true)`, fixture.tenantB.String()); err != nil {
		t.Fatalf("set RLS tenant: %v", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO purchase_orders(tenant_id,po_number,supplier_id,order_date,expected_delivery_date,currency,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,'2026-07-21','2026-07-22','IDR',$4,$4)`, fixture.tenantA, "PO-RLS-"+uuid.New().String(), fixture.supplier, fixture.buyer)
	var databaseError *pgconn.PgError
	if !errors.As(err, &databaseError) || databaseError.Code != "42501" {
		t.Fatalf("cross-tenant insert error = %T %v, want RLS 42501", err, err)
	}
}
