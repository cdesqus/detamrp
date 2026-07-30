package emailing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecisionEmailMigrationExtendsEmailLogType(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "database", "migrations", "015_email_decision_results.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := strings.ToLower(string(content))
	for _, expected := range []string{"drop constraint if exists email_logs_email_type_check", "'decision'", "add constraint email_logs_email_type_check"} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}
}
