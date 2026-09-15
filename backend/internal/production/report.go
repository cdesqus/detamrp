package production

import (
	"context"
	"time"

	"github.com/google/uuid"

	"order-stock/backend/internal/database"
)

// OrderReport is everything an execution document shows for one production
// order: its plan, what was produced, what it cost and what is still in WIP.
type OrderReport struct {
	Order       Order           `json:"order"`
	Cost        OrderCost       `json:"cost"`
	WIP         WIPOrderBalance `json:"wip"`
	GeneratedAt time.Time       `json:"generatedAt"`
}

func (s *Store) OrderReport(ctx context.Context, a Actor, id uuid.UUID) (r OrderReport, err error) {
	r.GeneratedAt = time.Now()
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		var e error
		if r.Order, e = loadOrder(ctx, tx, a, id); e != nil {
			return e
		}
		if r.Cost, e = scanOrderCost(tx.QueryRow(ctx, costSelect+` WHERE o.tenant_id=$1 AND o.id=$2`, a.TenantID, id)); e != nil {
			return e
		}
		if r.Cost.Operations, e = loadOperationCosts(ctx, tx, a, r.Cost); e != nil {
			return e
		}
		if r.Cost.Materials, e = loadMaterialCosts(ctx, tx, a, r.Cost); e != nil {
			return e
		}
		r.WIP, e = loadWIPOrder(ctx, tx, a, id, true)
		return e
	})
	return
}
