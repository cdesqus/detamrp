package rbac

import "testing"

func TestAllowsUnionOfPermissionsAcrossRoles(t *testing.T) {
	permissions := []string{"po.view", "po.create", "inventory.view"}
	if !Allows(permissions, "po.view", "inventory.view") {
		t.Fatal("expected permission union to allow request")
	}
}

func TestDeniesWhenAnyRequiredPermissionIsMissing(t *testing.T) {
	if Allows([]string{"po.view"}, "po.view", "po.approve") {
		t.Fatal("missing permission was allowed")
	}
}

func TestCatalogContainsPrototypeAdministrationAndOperationalPermissions(t *testing.T) {
	required := []string{"po.approve", "receiving.submit", "inventory.adjust_minus", "smtp_settings.test", "role.manage"}
	for _, permission := range required {
		if _, ok := Catalog[permission]; !ok {
			t.Fatalf("catalog missing %s", permission)
		}
	}
}
