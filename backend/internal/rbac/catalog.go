package rbac

var Catalog = map[string]string{
	"po.view": "View supplier orders", "po.create": "Create supplier orders", "po.edit_draft": "Edit draft supplier orders",
	"po.submit": "Submit supplier orders", "po.approve": "Approve supplier orders", "po.reject": "Reject supplier orders",
	"po.price.view": "View purchase prices", "po.unit_price.edit": "Edit purchase prices",
	"dn.view": "View delivery notes", "dn.issue": "Issue delivery notes", "dn.cancel": "Cancel delivery notes",
	"receiving.view": "View receiving", "receiving.create": "Create receiving", "receiving.submit": "Complete receiving",
	"inventory.view": "View inventory", "inventory.consume": "Consume inventory", "inventory.move": "Move inventory",
	"inventory.adjust_plus": "Increase inventory", "inventory.adjust_minus": "Decrease inventory", "inventory.stock_take": "Perform stock taking",
	"integration.view": "View integrations", "integration.retry": "Retry integrations",
	"smtp_settings.view": "View SMTP settings", "smtp_settings.manage": "Manage SMTP settings", "smtp_settings.test": "Test SMTP settings",
	"email_log.view": "View email log", "email_log.resend": "Resend emails",
	"user.manage": "Manage users", "role.manage": "Manage roles", "configuration.manage": "Manage configuration",
	"master_data.view": "View master data", "master_data.manage": "Manage master data",
}

func Allows(granted []string, required ...string) bool {
	set := make(map[string]struct{}, len(granted))
	for _, permission := range granted {
		set[permission] = struct{}{}
	}
	for _, permission := range required {
		if _, ok := set[permission]; !ok {
			return false
		}
	}
	return true
}
