package outgoing

import (
	"os"
	"strings"
	"testing"
)

func TestMigrationDefinesOutgoingAndRLS(t *testing.T) {
	b, e := os.ReadFile("../../../database/migrations/008_outgoing_material.sql")
	if e != nil {
		t.Fatal(e)
	}
	for _, x := range []string{"outgoing_sessions", "outgoing_session_scans", "outgoing_documents", "ENABLE ROW LEVEL SECURITY"} {
		if !strings.Contains(string(b), x) {
			t.Errorf("missing %s", x)
		}
	}
}

func TestOptionalDestinationMigration(t *testing.T) {
	b, err := os.ReadFile("../../../database/migrations/009_outgoing_optional_destination.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(b))
	for _, required := range []string{
		"drop constraint if exists outgoing_sessions_destination_check",
		"check (length(trim(destination)) <= 120)",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("missing %q", required)
		}
	}
}
