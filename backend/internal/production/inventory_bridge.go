package production

import (
	"context"

	"github.com/google/uuid"

	"order-stock/backend/internal/database"
)

// Material leaves the warehouse when production books it, and comes back when
// that booking is corrected or voided. Production moves quantities rather than
// whole kanban lots, so these ledger rows carry no lot reference.
const inventoryLedgerInsert = `INSERT INTO inventory_ledger_entries
 (tenant_id,event_type,raw_material_id,quantity_delta,base_unit_code,warehouse,location,reference_type,reference_id,created_by_user_id)`

// issueMaterialToInventory takes the usage of one daily entry out of stock.
func issueMaterialToInventory(ctx context.Context, tx database.TenantTx, a Actor, entryID uuid.UUID, materials []MaterialUsage) error {
	for _, material := range materials {
		if !material.Quantity.IsPositive() {
			continue
		}
		if _, e := tx.Exec(ctx, inventoryLedgerInsert+`
 VALUES($1,'PRODUCTION_ISSUE',$2,-$3,$4,'RAW MATERIAL','PRODUCTION','PRODUCTION_ENTRY',$5,$6)`,
			a.TenantID, material.MaterialID, material.Quantity, material.UnitCode, entryID, a.UserID); e != nil {
			return e
		}
	}
	return nil
}

// returnMaterialToInventory gives back whatever an entry still holds. It posts
// the outstanding balance per material, so running it twice changes nothing.
func returnMaterialToInventory(ctx context.Context, tx database.TenantTx, a Actor, entryID uuid.UUID) error {
	_, e := tx.Exec(ctx, inventoryLedgerInsert+`
 SELECT $1,'PRODUCTION_RETURN',l.raw_material_id,-sum(l.quantity_delta),max(l.base_unit_code),'RAW MATERIAL','PRODUCTION','PRODUCTION_ENTRY',$2,$3
 FROM inventory_ledger_entries l
 WHERE l.tenant_id=$1 AND l.reference_type='PRODUCTION_ENTRY' AND l.reference_id=$2
 GROUP BY l.raw_material_id HAVING sum(l.quantity_delta)<0`, a.TenantID, entryID, a.UserID)
	return e
}
