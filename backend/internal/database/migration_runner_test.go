package database

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestComposeRunsVersionedMigrationsBeforeBackend(t *testing.T) {
	content := readRepositoryFile(t, "docker-compose.yml")
	for _, fragment := range []string{
		"migrate:",
		"condition: service_completed_successfully",
		"PGUSER: nextgen",
		"PGPASSWORD: nextgen",
		"./database/migrations:/migrations:ro",
		"./database/migration-bootstrap.sql:/migration-tools/migration-bootstrap.sql:ro",
		`sed 's/\\r$//' /migration-tools/migrate.sh`,
	} {
		if !strings.Contains(content, fragment) {
			t.Errorf("compose migration contract missing %q", fragment)
		}
	}
	if strings.Contains(content, "/docker-entrypoint-initdb.d") {
		t.Fatal("postgres init directory must not bypass the versioned migration runner")
	}
}

func TestLiveSchemaMigrationsAreAppliedExactlyOnce(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer pool.Close()
	var count, distinct, minimum, maximum int
	if err := pool.QueryRow(context.Background(), `SELECT count(*),count(DISTINCT version),min(version),max(version) FROM schema_migrations`).Scan(&count, &distinct, &minimum, &maximum); err != nil {
		t.Fatalf("read schema migrations: %v", err)
	}
	if count != 9 || distinct != 9 || minimum != 1 || maximum != 9 {
		t.Fatalf("schema migration versions = count %d distinct %d range %d-%d, want nine unique versions 1-9", count, distinct, minimum, maximum)
	}
}

func TestMigrationScriptBaselinesOnlyLegacyVersionsAndSortsNumericFiles(t *testing.T) {
	content := readRepositoryFile(t, "database", "migrate.sh")
	for _, fragment := range []string{
		"migration-bootstrap.sql",
		"sort -n",
		"migrate-one.sql",
	} {
		if !strings.Contains(content, fragment) {
			t.Errorf("migration runner missing %q", fragment)
		}
	}
	bootstrap := readRepositoryFile(t, "database", "migration-bootstrap.sql")
	for _, fragment := range []string{
		"partial legacy schema",
		"measurements", "suppliers", "raw_materials", "warehouses", "warehouse_locations",
		"standard_unit_price", "currency", "default_approver_user_id", "users_tenant_email_ci_key",
		"(1, '001_foundation.sql')", "(2, '002_master_data.sql')", "(3, '003_raw_material_price.sql')", "(4, '004_settings_identity.sql')",
	} {
		if !strings.Contains(bootstrap, fragment) {
			t.Errorf("migration bootstrap missing sentinel %q", fragment)
		}
	}
	if strings.Contains(bootstrap, "005_purchase_orders.sql") {
		t.Fatal("legacy bootstrap must not mark migration 005 as applied")
	}
}

func TestLiveMigrationBootstrapRejectsPartialLegacySchemaWithoutLedger(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer pool.Close()
	connection, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire connection: %v", err)
	}
	defer connection.Release()
	schema := "migration_partial_" + strings.ReplaceAll(filepath.Base(t.TempDir()), "-", "_")
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := connection.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatalf("create partial schema: %v", err)
	}
	defer connection.Exec(ctx, "DROP SCHEMA IF EXISTS "+identifier+" CASCADE")
	if _, err := connection.Exec(ctx, "SET search_path TO "+identifier); err != nil {
		t.Fatalf("set partial schema: %v", err)
	}
	if _, err := connection.Exec(ctx, `CREATE TABLE tenants(id uuid PRIMARY KEY)`); err != nil {
		t.Fatalf("create tenants sentinel: %v", err)
	}
	bootstrap := readRepositoryFile(t, "database", "migration-bootstrap.sql")
	_, err = connection.Exec(ctx, bootstrap, pgx.QueryExecModeSimpleProtocol)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "partial legacy schema") {
		t.Fatalf("bootstrap error = %v, want clear partial legacy failure", err)
	}
	if _, rollbackErr := connection.Exec(ctx, "ROLLBACK"); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
		t.Fatalf("rollback failed bootstrap: %v", rollbackErr)
	}
	var ledgerTables int
	if err := connection.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema=$1 AND table_name='schema_migrations'`, schema).Scan(&ledgerTables); err != nil {
		t.Fatalf("inspect migration ledger: %v", err)
	}
	if ledgerTables != 0 {
		t.Fatal("partial legacy bootstrap persisted a migration ledger")
	}
}

func TestMigrationTransactionAppliesEachVersionExactlyOnce(t *testing.T) {
	content := readRepositoryFile(t, "database", "migrate-one.sql")
	for _, fragment := range []string{
		"pg_advisory_xact_lock",
		"NOT EXISTS (SELECT 1 FROM schema_migrations WHERE version = :migration_version)",
		`\if :apply_migration`,
		`\i :migration_file`,
		"INSERT INTO schema_migrations(version, name)",
	} {
		if !strings.Contains(content, fragment) {
			t.Errorf("single-migration transaction missing %q", fragment)
		}
	}
}

func readRepositoryFile(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{"..", "..", ".."}, parts...)...)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}
