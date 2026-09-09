package salesmaster

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/database"
)

func TestSalesMasterMigrationDefinesTenantScopedMastersPermissionsAndAudit(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "database", "migrations", "021_sales_masters.sql"))
	if err != nil {
		t.Fatalf("read sales master migration: %v", err)
	}
	sql := strings.ToLower(string(content))
	for _, fragment := range []string{
		"create table customers", "create table finished_goods", "unique (tenant_id, code)",
		"unique (tenant_id, item_code)", "sales_price numeric(20,6)", "check (sales_price > 0)",
		"foreign key (tenant_id, base_unit_id) references units(tenant_id, id)",
		"alter table customers force row level security", "alter table finished_goods force row level security",
		"customer.view", "customer.manage", "fg.view", "fg.manage", "fg.price.manage",
		"where r.code = 'admin'", "create trigger activity_audit", "grant select, insert, update on customers, finished_goods to nextgen_app",
	} {
		if !strings.Contains(sql, fragment) {
			t.Errorf("migration missing %q", fragment)
		}
	}
}

func TestSQLStoreScopesDuplicateFinishedGoodItemCodesByTenant(t *testing.T) {
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatalf("open admin pool: %v", err)
	}
	defer admin.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	userA, userB := uuid.New(), uuid.New()
	unitA, unitB := uuid.New(), uuid.New()
	if _, err := admin.Exec(ctx, `INSERT INTO tenants(id,code,name) VALUES($1,$2,'Sales A'),($3,$4,'Sales B')`,
		tenantA, "SALES_A_"+tenantA.String(), tenantB, "SALES_B_"+tenantB.String()); err != nil {
		t.Fatalf("insert tenants: %v", err)
	}
	if _, err := admin.Exec(ctx, `INSERT INTO users(id,tenant_id,username,display_name,email,password_hash) VALUES
 ($1,$2,'sales-a','Sales A','sales-a@test.invalid','x'),($3,$4,'sales-b','Sales B','sales-b@test.invalid','x')`,
		userA, tenantA, userB, tenantB); err != nil {
		t.Fatalf("insert users: %v", err)
	}
	if _, err := admin.Exec(ctx, `INSERT INTO units(id,tenant_id,code,name,created_by_user_id,updated_by_user_id) VALUES
 ($1,$2,'PCS','Pieces',$3,$3),($4,$5,'PCS','Pieces',$6,$6)`, unitA, tenantA, userA, unitB, tenantB, userB); err != nil {
		t.Fatalf("insert fixture: %v", err)
	}
	defer func() {
		ids := []uuid.UUID{tenantA, tenantB}
		if _, cleanupErr := admin.Exec(ctx, `ALTER TABLE activity_logs DISABLE TRIGGER activity_logs_append_only`); cleanupErr != nil {
			t.Errorf("disable activity log guard: %v", cleanupErr)
			return
		}
		defer func() {
			if _, cleanupErr := admin.Exec(ctx, `ALTER TABLE activity_logs ENABLE TRIGGER activity_logs_append_only`); cleanupErr != nil {
				t.Errorf("restore activity log guard: %v", cleanupErr)
			}
		}()
		for _, statement := range []string{
			`DELETE FROM activity_logs WHERE tenant_id=ANY($1)`,
			`DELETE FROM finished_goods WHERE tenant_id=ANY($1)`,
			`DELETE FROM activity_logs WHERE tenant_id=ANY($1)`,
			`DELETE FROM units WHERE tenant_id=ANY($1)`,
			`DELETE FROM users WHERE tenant_id=ANY($1)`,
			`DELETE FROM activity_logs WHERE tenant_id=ANY($1)`,
			`DELETE FROM tenants WHERE id=ANY($1)`,
		} {
			if _, cleanupErr := admin.Exec(ctx, statement, ids); cleanupErr != nil {
				t.Errorf("cleanup fixture %q: %v", statement, cleanupErr)
				return
			}
		}
	}()

	appDB, err := database.Open(ctx, salesMasterApplicationURL(t, adminURL))
	if err != nil {
		t.Fatalf("open application pool: %v", err)
	}
	defer appDB.Close()
	store := NewStore(appDB)
	inputA := FinishedGoodInput{ItemCode: "SHARED-FG", Name: "Tenant A FG", BaseUnitID: unitA, SalesPrice: decimal.NewFromInt(10), Currency: "IDR"}
	inputB := FinishedGoodInput{ItemCode: "SHARED-FG", Name: "Tenant B FG", BaseUnitID: unitB, SalesPrice: decimal.NewFromInt(20), Currency: "IDR"}

	if _, err := store.CreateFinishedGood(ctx, Actor{TenantID: tenantA, UserID: userA}, inputA); err != nil {
		t.Fatalf("create tenant A finished good: %v", err)
	}
	if _, err := store.CreateFinishedGood(ctx, Actor{TenantID: tenantB, UserID: userB}, inputB); err != nil {
		t.Fatalf("same code in tenant B: %v", err)
	}
	if _, err := store.CreateFinishedGood(ctx, Actor{TenantID: tenantA, UserID: userA}, inputA); err == nil {
		t.Fatal("duplicate Item Code in tenant A unexpectedly succeeded")
	} else {
		var conflict ConflictError
		if !errors.As(err, &conflict) || conflict.Fields["itemCode"] == "" {
			t.Fatalf("duplicate error = %T %v, want itemCode conflict", err, err)
		}
	}
}

func salesMasterApplicationURL(t *testing.T, raw string) string {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	if parsed.Scheme == "postgres" || parsed.Scheme == "postgresql" {
		if parsed.User == nil {
			parsed.User = url.UserPassword("nextgen_app", "nextgen_app")
		} else {
			parsed.User = url.UserPassword("nextgen_app", "nextgen_app")
		}
		return parsed.String()
	}
	t.Fatalf("TEST_DATABASE_URL must be a PostgreSQL URL, got %q", raw)
	return fmt.Sprint(raw)
}
