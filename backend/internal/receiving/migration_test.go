package receiving

import (
	"os"
	"strings"
	"testing"
)

func TestReceivingMigrationDefinesExclusiveSessionsLedgerAndRLS(t *testing.T) {
	b, err := os.ReadFile("../../../database/migrations/007_receiving_inventory.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, required := range []string{"receiving_sessions", "receiving_session_scans", "inventory_ledger_entries", "integration_outbox", "PARTIALLY_RECEIVED", "FULLY_RECEIVED", "ENABLE ROW LEVEL SECURITY", "one_open_receiving_session_per_dn"} {
		if !strings.Contains(sql, required) {
			t.Errorf("migration missing %s", required)
		}
	}
}
