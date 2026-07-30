package masterdata

import (
	"os"
	"strings"
	"testing"
)

func TestReferenceStoreUsesTenantScopedTablesAndAuditJoins(t *testing.T) {
	content, err := os.ReadFile("reference_store.go")
	if err != nil {
		t.Fatalf("read reference store: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{
		"FROM categories c",
		"FROM packings p",
		"FROM plants p",
		"created_by_user_id",
		"updated_by_user_id",
		".tenant_id=$1",
		"ILIKE '%'||$2||'%'",
		"masterDataWriteError",
	} {
		if !strings.Contains(source, fragment) {
			t.Errorf("reference store missing %q", fragment)
		}
	}
}
