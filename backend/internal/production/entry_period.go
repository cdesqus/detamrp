package production

import (
	"context"
	"order-stock/backend/internal/database"
	"strings"
	"time"
)

type ProductionPeriod struct {
	Period   string `json:"period"`
	Closed   bool   `json:"closed"`
	ClosedBy string `json:"closedBy"`
	Reason   string `json:"reason"`
}

func lockEntryPeriod(ctx context.Context, tx database.TenantTx, a Actor, date string) error {
	period := date[:7]
	if _, e := tx.Exec(ctx, `INSERT INTO production_periods(tenant_id,period) VALUES($1,$2) ON CONFLICT DO NOTHING`, a.TenantID, period); e != nil {
		return e
	}
	var closed bool
	if e := tx.QueryRow(ctx, `SELECT closed_at IS NOT NULL FROM production_periods WHERE tenant_id=$1 AND period=$2 FOR SHARE`, a.TenantID, period).Scan(&closed); e != nil {
		return e
	}
	if closed {
		return invalid("Production period %s is closed", period)
	}
	return nil
}
func (s *Store) ListProductionPeriods(ctx context.Context, a Actor) (items []ProductionPeriod, err error) {
	items = []ProductionPeriod{}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT p.period,p.closed_at IS NOT NULL,COALESCE(u.display_name,''),p.close_reason FROM production_periods p LEFT JOIN users u ON u.tenant_id=p.tenant_id AND u.id=p.closed_by_user_id WHERE p.tenant_id=$1 ORDER BY period DESC`, a.TenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var p ProductionPeriod
			if e = rows.Scan(&p.Period, &p.Closed, &p.ClosedBy, &p.Reason); e != nil {
				return e
			}
			items = append(items, p)
		}
		return rows.Err()
	})
	return
}
func (s *Store) CloseProductionPeriod(ctx context.Context, a Actor, period, reason string) error {
	if len(period) != 7 {
		return invalid("Use a valid YYYY-MM period")
	}
	if _, e := time.Parse("2006-01", period); e != nil {
		return invalid("Use a valid YYYY-MM period")
	}
	if strings.TrimSpace(reason) == "" || len(reason) > 2000 {
		return invalid("A close reason is required, up to 2000 characters")
	}
	return s.transaction(ctx, a, func(tx database.TenantTx) error {
		if _, e := tx.Exec(ctx, `INSERT INTO production_periods(tenant_id,period) VALUES($1,$2) ON CONFLICT DO NOTHING`, a.TenantID, period); e != nil {
			return e
		}
		var closed bool
		if e := tx.QueryRow(ctx, `SELECT closed_at IS NOT NULL FROM production_periods WHERE tenant_id=$1 AND period=$2 FOR UPDATE`, a.TenantID, period).Scan(&closed); e != nil {
			return e
		}
		if closed {
			return invalid("This period is already closed")
		}
		_, e := tx.Exec(ctx, `UPDATE production_periods SET closed_at=now(),closed_by_user_id=$3,close_reason=$4 WHERE tenant_id=$1 AND period=$2`, a.TenantID, period, a.UserID, strings.TrimSpace(reason))
		return e
	})
}
