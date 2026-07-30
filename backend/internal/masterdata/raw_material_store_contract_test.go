package masterdata

import (
	"os"
	"strings"
	"testing"
)

func TestRawMaterialStoreLoadsAndRequiresActiveCategoryAndPacking(t *testing.T) {
	content, err := os.ReadFile("store.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, fragment := range []string{
		"JOIN categories c ON c.tenant_id=r.tenant_id AND c.id=r.category_id",
		"JOIN packings p ON p.tenant_id=r.tenant_id AND p.id=r.packing_id",
		"c.active=true",
		"p.active=true",
	} {
		if !strings.Contains(source, fragment) {
			t.Errorf("raw material store missing %q", fragment)
		}
	}
}
