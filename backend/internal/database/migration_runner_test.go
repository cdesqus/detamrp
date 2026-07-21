package database

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	if count != 5 || distinct != 5 || minimum != 1 || maximum != 5 {
		t.Fatalf("schema migration versions = count %d distinct %d range %d-%d, want five unique versions 1-5", count, distinct, minimum, maximum)
	}
}

func TestMigrationScriptBaselinesOnlyLegacyVersionsAndSortsNumericFiles(t *testing.T) {
	content := readRepositoryFile(t, "database", "migrate.sh")
	for _, fragment := range []string{
		"to_regclass('public.tenants')",
		"001_foundation.sql",
		"002_master_data.sql",
		"003_raw_material_price.sql",
		"004_settings_identity.sql",
		"sort -n",
		"migrate-one.sql",
	} {
		if !strings.Contains(content, fragment) {
			t.Errorf("migration runner missing %q", fragment)
		}
	}
	baseline := content[strings.Index(content, "legacy_initialized"):strings.Index(content, "find /migrations")]
	if strings.Contains(baseline, "005_purchase_orders.sql") {
		t.Fatal("legacy baseline must not mark migration 005 as applied")
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
