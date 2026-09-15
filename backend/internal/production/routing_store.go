package production

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"order-stock/backend/internal/database"
)

const routingSelect = `SELECT r.id,COALESCE(r.finished_good_id,r.raw_material_id),
 CASE WHEN r.finished_good_id IS NOT NULL THEN 'FG' ELSE 'RAW_MATERIAL' END,
 COALESCE(f.item_code,m.code,''),COALESCE(f.name,m.name,''),COALESCE(fu.code,mu.code,''),
 r.name,r.currency,r.revision,r.active,r.steps,r.updated_at,u.display_name,
 (SELECT count(*) FROM production_orders o WHERE o.tenant_id=r.tenant_id AND o.routing_id=r.id)
 FROM production_routings r
 LEFT JOIN finished_goods f ON f.tenant_id=r.tenant_id AND f.id=r.finished_good_id
 LEFT JOIN units fu ON fu.tenant_id=f.tenant_id AND fu.id=f.base_unit_id
 LEFT JOIN raw_materials m ON m.tenant_id=r.tenant_id AND m.id=r.raw_material_id
 LEFT JOIN units mu ON mu.tenant_id=m.tenant_id AND mu.id=m.base_unit_id
 JOIN users u ON u.tenant_id=r.tenant_id AND u.id=r.created_by_user_id`

const nilUUID = "00000000-0000-0000-0000-000000000000"

func scanRouting(row pgx.Row) (r Routing, err error) {
	var raw []byte
	if err = row.Scan(&r.ID, &r.PartID, &r.Kind, &r.PartNumber, &r.PartName, &r.UnitCode, &r.Name, &r.Currency,
		&r.Revision, &r.Active, &raw, &r.UpdatedAt, &r.CreatedBy, &r.LinkedOrders); err != nil {
		return
	}
	r.Steps = []RoutingStep{}
	r.History = []PlanHistory{}
	if err = json.Unmarshal(raw, &r.Steps); err != nil {
		return
	}
	r.TotalRate = decimal.Zero
	for _, step := range r.Steps {
		r.TotalRate = r.TotalRate.Add(step.Rate)
	}
	return
}

func (s *Store) ListRoutings(ctx context.Context, a Actor) (items []Routing, err error) {
	items = []Routing{}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, routingSelect+` WHERE r.tenant_id=$1 ORDER BY COALESCE(f.item_code,m.code),r.revision DESC`, a.TenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			r, e := scanRouting(rows)
			if e != nil {
				return e
			}
			items = append(items, r)
		}
		return rows.Err()
	})
	return
}

func (s *Store) GetRouting(ctx context.Context, a Actor, id uuid.UUID) (r Routing, err error) {
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		var e error
		r, e = loadRouting(ctx, tx, a, id)
		return e
	})
	return
}

func loadRouting(ctx context.Context, tx database.TenantTx, a Actor, id uuid.UUID) (r Routing, err error) {
	if r, err = scanRouting(tx.QueryRow(ctx, routingSelect+` WHERE r.tenant_id=$1 AND r.id=$2`, a.TenantID, id)); err != nil {
		return
	}
	r.History, err = loadHistory(ctx, tx, a, "production_routings", id)
	return
}

// RoutingOptions lists the active parts a routing may be defined for.
func (s *Store) RoutingOptions(ctx context.Context, a Actor) (out RoutingOptions, err error) {
	masters, err := s.Options(ctx, a)
	return RoutingOptions{Parts: masters.Parts}, err
}

// outputColumns maps an output kind onto the exclusive routing columns.
func outputColumns(kind string, part uuid.UUID) (fg, rm uuid.UUID) {
	if kind == "FG" {
		return part, uuid.Nil
	}
	return uuid.Nil, part
}

func resolveRoutingPart(ctx context.Context, tx database.TenantTx, a Actor, kind string, part uuid.UUID) error {
	query := `SELECT 1 FROM finished_goods f JOIN units u ON u.tenant_id=f.tenant_id AND u.id=f.base_unit_id WHERE f.tenant_id=$1 AND f.id=$2 AND f.active AND u.active`
	if kind != "FG" {
		query = `SELECT 1 FROM raw_materials m JOIN units u ON u.tenant_id=m.tenant_id AND u.id=m.base_unit_id WHERE m.tenant_id=$1 AND m.id=$2 AND m.active AND u.active`
	}
	var found int
	if e := tx.QueryRow(ctx, query, a.TenantID, part).Scan(&found); e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrInvalidReference
		}
		return e
	}
	return nil
}

// Lock the stable output row before any routing row, matching the master-before-
// routing order used when creating production orders. Existing revisions alone
// cannot serialize the first creation or protect against a stale SELECT snapshot.
func lockRoutingPart(ctx context.Context, tx database.TenantTx, a Actor, kind string, part uuid.UUID) error {
	query := `SELECT id FROM finished_goods WHERE tenant_id=$1 AND id=$2 FOR UPDATE`
	if kind != "FG" {
		query = `SELECT id FROM raw_materials WHERE tenant_id=$1 AND id=$2 FOR UPDATE`
	}
	var id uuid.UUID
	return tx.QueryRow(ctx, query, a.TenantID, part).Scan(&id)
}

// The caller holds the part lock, so this fresh statement sees every revision
// committed by the previous holder before assigning the next number.
func lockRoutingFamily(ctx context.Context, tx database.TenantTx, a Actor, fg, rm uuid.UUID) (activeCount, nextRevision int, err error) {
	rows, e := tx.Query(ctx, `SELECT revision,active FROM production_routings WHERE tenant_id=$1 AND (finished_good_id=NULLIF($2::uuid,'`+nilUUID+`') OR raw_material_id=NULLIF($3::uuid,'`+nilUUID+`')) ORDER BY id FOR UPDATE`, a.TenantID, fg, rm)
	if e != nil {
		return 0, 0, e
	}
	defer rows.Close()
	nextRevision = 1
	for rows.Next() {
		var revision int
		var active bool
		if e = rows.Scan(&revision, &active); e != nil {
			return 0, 0, e
		}
		if active {
			activeCount++
		}
		if revision >= nextRevision {
			nextRevision = revision + 1
		}
	}
	return activeCount, nextRevision, rows.Err()
}

func (s *Store) CreateRouting(ctx context.Context, a Actor, input RoutingInput) (r Routing, err error) {
	input.Normalize()
	if e := input.Validate(true); e != nil {
		return r, e
	}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		if e := lockRoutingPart(ctx, tx, a, input.Kind, input.PartID); e != nil {
			return e
		}
		if e := resolveRoutingPart(ctx, tx, a, input.Kind, input.PartID); e != nil {
			return e
		}
		fg, rm := outputColumns(input.Kind, input.PartID)
		activeCount, revision, e := lockRoutingFamily(ctx, tx, a, fg, rm)
		if e != nil {
			return e
		}
		steps, e := json.Marshal(input.Steps)
		if e != nil {
			return e
		}
		id := uuid.New()
		// The first routing of a part goes live immediately; later revisions stay
		// inactive until they are explicitly activated.
		if _, e = tx.Exec(ctx, `INSERT INTO production_routings(id,tenant_id,finished_good_id,raw_material_id,name,revision,currency,steps,active,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,NULLIF($3::uuid,'`+nilUUID+`'),NULLIF($4::uuid,'`+nilUUID+`'),$5,$6,$7,$8,$9,$10,$10)`,
			id, a.TenantID, fg, rm, input.Name, revision, input.Currency, steps, activeCount == 0, a.UserID); e != nil {
			return e
		}
		r, e = loadRouting(ctx, tx, a, id)
		return e
	})
	return
}

func (s *Store) UpdateRouting(ctx context.Context, a Actor, id uuid.UUID, input RoutingInput) (r Routing, err error) {
	input.Normalize()
	if e := input.Validate(false); e != nil {
		return r, e
	}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		current, e := lockRouting(ctx, tx, a, id)
		if e != nil {
			return e
		}
		if !canEditRouting(current.LinkedOrders) {
			return ErrConflict
		}
		steps, e := json.Marshal(input.Steps)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE production_routings SET name=$3,currency=$4,steps=$5,updated_at=now(),updated_by_user_id=$6 WHERE tenant_id=$1 AND id=$2`,
			a.TenantID, id, input.Name, input.Currency, steps, a.UserID); e != nil {
			return e
		}
		r, e = loadRouting(ctx, tx, a, id)
		return e
	})
	return
}

func lockRouting(ctx context.Context, tx database.TenantTx, a Actor, id uuid.UUID) (Routing, error) {
	var kind string
	var part uuid.UUID
	if e := tx.QueryRow(ctx, `SELECT CASE WHEN finished_good_id IS NOT NULL THEN 'FG' ELSE 'RAW_MATERIAL' END,COALESCE(finished_good_id,raw_material_id) FROM production_routings WHERE tenant_id=$1 AND id=$2`, a.TenantID, id).Scan(&kind, &part); e != nil {
		return Routing{}, e
	}
	if e := lockRoutingPart(ctx, tx, a, kind, part); e != nil {
		return Routing{}, e
	}
	var exists uuid.UUID
	if e := tx.QueryRow(ctx, `SELECT id FROM production_routings WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, a.TenantID, id).Scan(&exists); e != nil {
		return Routing{}, e
	}
	return loadRouting(ctx, tx, a, id)
}

// RoutingAction applies the activate, deactivate and delete transitions.
func (s *Store) RoutingAction(ctx context.Context, a Actor, id uuid.UUID, action string) (r Routing, err error) {
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		current, e := lockRouting(ctx, tx, a, id)
		if e != nil {
			return e
		}
		fg, rm := outputColumns(current.Kind, current.PartID)
		switch action {
		case "activate":
			if _, _, e = lockRoutingFamily(ctx, tx, a, fg, rm); e != nil {
				return e
			}
			if e = resolveRoutingPart(ctx, tx, a, current.Kind, current.PartID); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `UPDATE production_routings SET active=false,updated_at=now(),updated_by_user_id=$3 WHERE tenant_id=$1 AND active AND id<>$2 AND (finished_good_id=NULLIF($4::uuid,'`+nilUUID+`') OR raw_material_id=NULLIF($5::uuid,'`+nilUUID+`'))`,
				a.TenantID, id, a.UserID, fg, rm); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `UPDATE production_routings SET active=true,updated_at=now(),updated_by_user_id=$3 WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, a.UserID); e != nil {
				return e
			}
		case "deactivate":
			if !current.Active {
				return ErrConflict
			}
			if _, e = tx.Exec(ctx, `UPDATE production_routings SET active=false,updated_at=now(),updated_by_user_id=$3 WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, a.UserID); e != nil {
				return e
			}
		case "delete":
			if current.LinkedOrders > 0 {
				return ErrConflict
			}
			_, e = tx.Exec(ctx, `DELETE FROM production_routings WHERE tenant_id=$1 AND id=$2`, a.TenantID, id)
			return e
		default:
			return invalid("Unsupported routing action")
		}
		r, e = loadRouting(ctx, tx, a, id)
		return e
	})
	return
}
