package production

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"order-stock/backend/internal/database"
)

var ErrConflict = errors.New("planning action is not allowed in its current state")
var ErrNotFound = errors.New("planning not found")
var ErrInvalidReference = errors.New("plant and parts must exist, be active, and belong to your company")

type Store struct{ db database.TenantBeginner }

func NewStore(db database.TenantBeginner) *Store { return &Store{db: db} }
func (s *Store) transaction(ctx context.Context, a Actor, fn func(database.TenantTx) error) error {
	err := database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, fn)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

const planSelect = `SELECT p.id,p.plan_number,p.period_start,p.period_end,COALESCE(p.plant_id,'00000000-0000-0000-0000-000000000000'),p.status,p.notes,COALESCE(pl.name,''),u.display_name,p.updated_at,
 (SELECT count(*) FROM production_plan_lines l WHERE l.tenant_id=p.tenant_id AND l.plan_id=p.id),
 (SELECT COALESCE(sum(planned_qty),0) FROM production_plan_lines l WHERE l.tenant_id=p.tenant_id AND l.plan_id=p.id),
 (SELECT count(*) FROM production_orders o WHERE o.tenant_id=p.tenant_id AND o.plan_id=p.id)
 FROM production_plans p LEFT JOIN plants pl ON pl.tenant_id=p.tenant_id AND pl.id=p.plant_id JOIN users u ON u.tenant_id=p.tenant_id AND u.id=p.created_by_user_id`

func scanPlan(row pgx.Row) (p Plan, err error) {
	err = row.Scan(&p.ID, &p.PlanNumber, &p.PeriodStart, &p.PeriodEnd, &p.PlantID, &p.Status, &p.Notes, &p.PlantName, &p.CreatedBy, &p.UpdatedAt, &p.TotalPart, &p.TotalPlannedQty, &p.LinkedProductionOrders)
	return
}
func (s *Store) List(ctx context.Context, a Actor) (items []Plan, err error) {
	items = []Plan{}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, planSelect+` WHERE p.tenant_id=$1 ORDER BY p.updated_at DESC,p.id`, a.TenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			p, e := scanPlan(rows)
			if e != nil {
				return e
			}
			items = append(items, p)
		}
		return rows.Err()
	})
	return
}
func (s *Store) Get(ctx context.Context, a Actor, id uuid.UUID) (p Plan, err error) {
	err = s.transaction(ctx, a, func(tx database.TenantTx) error { var e error; p, e = loadPlan(ctx, tx, a, id); return e })
	return
}
func loadPlan(ctx context.Context, tx database.TenantTx, a Actor, id uuid.UUID) (p Plan, err error) {
	p, err = scanPlan(tx.QueryRow(ctx, planSelect+` WHERE p.tenant_id=$1 AND p.id=$2`, a.TenantID, id))
	if err != nil {
		return
	}
	p.Lines = []PlanLine{}
	p.Orders = []LinkedOrder{}
	p.History = []PlanHistory{}
	rows, e := tx.Query(ctx, `SELECT id,COALESCE(finished_good_id,'00000000-0000-0000-0000-000000000000'),COALESCE(raw_material_id,'00000000-0000-0000-0000-000000000000'),planned_qty,unit_code,part_number,part_name FROM production_plan_lines WHERE tenant_id=$1 AND plan_id=$2 ORDER BY sort_position,id`, a.TenantID, id)
	if e != nil {
		return p, e
	}
	for rows.Next() {
		var l PlanLine
		if e = rows.Scan(&l.ID, &l.FinishedGoodID, &l.RawMaterialID, &l.PlannedQty, &l.UnitCode, &l.PartNumber, &l.PartName); e != nil {
			rows.Close()
			return p, e
		}
		p.Lines = append(p.Lines, l)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return p, e
	}
	rows, e = tx.Query(ctx, `SELECT o.id,o.order_number,o.status,o.planned_qty,COALESCE((SELECT sum(e.qty_good) FROM production_entries e JOIN production_operations op ON op.tenant_id=e.tenant_id AND op.id=e.operation_id WHERE e.voided_at IS NULL AND op.tenant_id=o.tenant_id AND op.production_order_id=o.id AND op.sequence_no=(SELECT max(last_op.sequence_no) FROM production_operations last_op WHERE last_op.tenant_id=o.tenant_id AND last_op.production_order_id=o.id)),0) FROM production_orders o WHERE o.tenant_id=$1 AND o.plan_id=$2 ORDER BY o.order_number`, a.TenantID, id)
	if e != nil {
		return p, e
	}
	for rows.Next() {
		var o LinkedOrder
		if e = rows.Scan(&o.ID, &o.OrderNumber, &o.Status, &o.PlannedQty, &o.GoodQty); e != nil {
			rows.Close()
			return p, e
		}
		p.Orders = append(p.Orders, o)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return p, e
	}
	rows, e = tx.Query(ctx, `SELECT action,actor_name,occurred_at FROM activity_logs WHERE tenant_id=$1 AND ((target_type='production_plans' AND target_id=$2) OR (target_type IN ('production_plan_lines','production_orders') AND COALESCE(after_data->>'plan_id',before_data->>'plan_id')=$2::text)) ORDER BY occurred_at DESC,id DESC`, a.TenantID, id)
	if e != nil {
		return p, e
	}
	defer rows.Close()
	for rows.Next() {
		var h PlanHistory
		if e = rows.Scan(&h.Action, &h.Actor, &h.OccurredAt); e != nil {
			return p, e
		}
		p.History = append(p.History, h)
	}
	return p, rows.Err()
}

// All mutations lock the parent before inspecting links or changing lines.
func lockPlan(ctx context.Context, tx database.TenantTx, a Actor, id uuid.UUID) (status PlanStatus, links int, err error) {
	err = tx.QueryRow(ctx, `SELECT status FROM production_plans WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, a.TenantID, id).Scan(&status)
	if err != nil {
		return
	}
	// A fresh statement snapshot sees orders committed while waiting for the lock.
	err = tx.QueryRow(ctx, `SELECT count(*) FROM production_orders WHERE tenant_id=$1 AND plan_id=$2`, a.TenantID, id).Scan(&links)
	return
}
func resolvePlan(ctx context.Context, tx database.TenantTx, a Actor, p *Plan) error {
	if err := ValidatePlanInput(*p); err != nil {
		return err
	}
	var plant uuid.UUID
	if e := tx.QueryRow(ctx, `SELECT id FROM plants WHERE tenant_id=$1 AND id=$2 AND active FOR SHARE`, a.TenantID, p.PlantID).Scan(&plant); e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrInvalidReference
		}
		return e
	}
	for i := range p.Lines {
		l := &p.Lines[i]
		q := `SELECT f.item_code,f.name,u.code FROM finished_goods f JOIN units u ON u.tenant_id=f.tenant_id AND u.id=f.base_unit_id WHERE f.tenant_id=$1 AND f.id=$2 AND f.active AND u.active FOR SHARE OF f,u`
		id := l.FinishedGoodID
		if id == uuid.Nil {
			q = `SELECT f.code,f.name,u.code FROM raw_materials f JOIN units u ON u.tenant_id=f.tenant_id AND u.id=f.base_unit_id WHERE f.tenant_id=$1 AND f.id=$2 AND f.active AND u.active FOR SHARE OF f,u`
			id = l.RawMaterialID
		}
		if e := tx.QueryRow(ctx, q, a.TenantID, id).Scan(&l.PartNumber, &l.PartName, &l.UnitCode); e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return ErrInvalidReference
			}
			return e
		}
	}
	return nil
}
func insertLines(ctx context.Context, tx database.TenantTx, a Actor, p Plan) error {
	for i, l := range p.Lines {
		if _, e := tx.Exec(ctx, `INSERT INTO production_plan_lines(tenant_id,plan_id,finished_good_id,raw_material_id,planned_qty,unit_code,sort_position,part_number,part_name) VALUES($1,$2,NULLIF($3::uuid,'00000000-0000-0000-0000-000000000000'),NULLIF($4::uuid,'00000000-0000-0000-0000-000000000000'),$5,$6,$7,$8,$9)`, a.TenantID, p.ID, l.FinishedGoodID, l.RawMaterialID, l.PlannedQty, l.UnitCode, i, l.PartNumber, l.PartName); e != nil {
			return e
		}
	}
	return nil
}
func (s *Store) Create(ctx context.Context, a Actor, p Plan) (result Plan, err error) {
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		if e := resolvePlan(ctx, tx, a, &p); e != nil {
			return e
		}
		p.ID = uuid.New()
		var sequence int64
		period := p.PeriodStart.Format("200601")
		if e := tx.QueryRow(ctx, `INSERT INTO production_plan_counters(tenant_id,period,last_number) VALUES($1,$2,1) ON CONFLICT(tenant_id,period) DO UPDATE SET last_number=production_plan_counters.last_number+1 RETURNING last_number`, a.TenantID, period).Scan(&sequence); e != nil {
			return e
		}
		p.PlanNumber = fmt.Sprintf("PP-%s-%06d", period, sequence)
		if _, e := tx.Exec(ctx, `INSERT INTO production_plans(id,tenant_id,plan_number,period_start,period_end,plant_id,notes,created_by_user_id,updated_by_user_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8)`, p.ID, a.TenantID, p.PlanNumber, p.PeriodStart, p.PeriodEnd, p.PlantID, p.Notes, a.UserID); e != nil {
			return e
		}
		if e := insertLines(ctx, tx, a, p); e != nil {
			return e
		}
		var e error
		result, e = loadPlan(ctx, tx, a, p.ID)
		return e
	})
	return
}
func (s *Store) Update(ctx context.Context, a Actor, id uuid.UUID, p Plan) (result Plan, err error) {
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		status, links, e := lockPlan(ctx, tx, a, id)
		if e != nil {
			return e
		}
		if status != PlanDraft || links > 0 {
			return ErrConflict
		}
		if e = resolvePlan(ctx, tx, a, &p); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE production_plans SET period_start=$3,period_end=$4,plant_id=$5,notes=$6,updated_by_user_id=$7,updated_at=now() WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, p.PeriodStart, p.PeriodEnd, p.PlantID, p.Notes, a.UserID); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `DELETE FROM production_plan_lines WHERE tenant_id=$1 AND plan_id=$2`, a.TenantID, id); e != nil {
			return e
		}
		p.ID = id
		if e = insertLines(ctx, tx, a, p); e != nil {
			return e
		}
		result, e = loadPlan(ctx, tx, a, id)
		return e
	})
	return
}
func (s *Store) Delete(ctx context.Context, a Actor, id uuid.UUID) error {
	return s.transaction(ctx, a, func(tx database.TenantTx) error {
		status, links, e := lockPlan(ctx, tx, a, id)
		if e != nil {
			return e
		}
		if status != PlanDraft || links > 0 {
			return ErrConflict
		}
		_, e = tx.Exec(ctx, `DELETE FROM production_plans WHERE tenant_id=$1 AND id=$2`, a.TenantID, id)
		return e
	})
}
func (s *Store) Approve(ctx context.Context, a Actor, id uuid.UUID) (Plan, error) {
	return s.transition(ctx, a, id, PlanDraft, PlanApproved)
}
func (s *Store) Close(ctx context.Context, a Actor, id uuid.UUID) (Plan, error) {
	return s.transition(ctx, a, id, PlanApproved, PlanClosed)
}
func (s *Store) transition(ctx context.Context, a Actor, id uuid.UUID, from, to PlanStatus) (result Plan, err error) {
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		status, links, e := lockPlan(ctx, tx, a, id)
		if e != nil {
			return e
		}
		if status != from || (to == PlanApproved && links > 0) {
			return ErrConflict
		}
		if to == PlanApproved {
			p, e := loadPlan(ctx, tx, a, id)
			if e != nil {
				return e
			}
			if e = resolvePlan(ctx, tx, a, &p); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `DELETE FROM production_plan_lines WHERE tenant_id=$1 AND plan_id=$2`, a.TenantID, id); e != nil {
				return e
			}
			if e = insertLines(ctx, tx, a, p); e != nil {
				return e
			}
		}
		if _, e = tx.Exec(ctx, `UPDATE production_plans SET status=$3,updated_by_user_id=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, to, a.UserID); e != nil {
			return e
		}
		result, e = loadPlan(ctx, tx, a, id)
		return e
	})
	return
}
func (s *Store) CreateOrders(ctx context.Context, a Actor, id uuid.UUID) (result Plan, err error) {
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		status, links, e := lockPlan(ctx, tx, a, id)
		if e != nil {
			return e
		}
		if status != PlanApproved || links > 0 {
			return ErrConflict
		}
		p, e := loadPlan(ctx, tx, a, id)
		if e != nil {
			return e
		}
		// Every line runs through the single order path so each order carries its
		// own BOM, routing and cost snapshot and consumes its planning allocation.
		due := p.PeriodEnd.Format("2006-01-02")
		for _, l := range p.Lines {
			if _, e = createOrder(ctx, tx, a, OrderInput{PlanLineID: l.ID, PlannedQty: l.PlannedQty, DueDate: due, Notes: p.Notes}); e != nil {
				return e
			}
		}
		result, e = loadPlan(ctx, tx, a, id)
		return e
	})
	return
}
