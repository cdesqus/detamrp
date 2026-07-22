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
