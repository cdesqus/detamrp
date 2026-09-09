package bom

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"order-stock/backend/internal/database"
)

type Store struct{ db *database.Pool }

func NewStore(db *database.Pool) *Store { return &Store{db} }
func tenant(a Actor) database.TenantContext {
	return database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}
}

const bomSelect = `SELECT b.id,b.output_kind,COALESCE(f.id,'00000000-0000-0000-0000-000000000000'),COALESCE(f.item_code,''),COALESCE(f.name,''),COALESCE(fu.code,''),COALESCE(r.id,'00000000-0000-0000-0000-000000000000'),COALESCE(r.code,''),COALESCE(r.name,''),COALESCE(ru.code,''),b.revision,b.status,b.notes FROM boms b LEFT JOIN finished_goods f ON f.tenant_id=b.tenant_id AND f.id=b.finished_good_id LEFT JOIN units fu ON fu.tenant_id=f.tenant_id AND fu.id=f.base_unit_id LEFT JOIN raw_materials r ON r.tenant_id=b.tenant_id AND r.id=b.raw_material_id LEFT JOIN units ru ON ru.tenant_id=r.tenant_id AND ru.id=r.base_unit_id`

func (s *Store) List(ctx context.Context, a Actor) (items []BOM, err error) {
	err = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, bomSelect+` WHERE b.tenant_id=$1 ORDER BY b.output_kind,COALESCE(f.item_code,r.code),b.revision DESC`, a.TenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			item, e := scanBOM(ctx, tx, rows, a.TenantID)
			if e != nil {
				return e
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return
}
func (s *Store) Get(ctx context.Context, a Actor, id uuid.UUID) (item BOM, err error) {
	err = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		loaded, e := scanBOM(ctx, tx, tx.QueryRow(ctx, bomSelect+` WHERE b.tenant_id=$1 AND b.id=$2`, a.TenantID, id), a.TenantID)
		if errors.Is(e, pgx.ErrNoRows) {
			return fmt.Errorf("BOM not found")
		}
		item = loaded
		return e
	})
	return
}
func scanBOM(ctx context.Context, tx database.TenantTx, row pgx.Row, tenantID uuid.UUID, target ...*BOM) (BOM, error) {
	var item BOM
	if len(target) > 0 {
		item = *target[0]
	}
	var fgID, rmID uuid.UUID
	var fgCode, fgName, fgUnit, rmCode, rmName, rmUnit string
	if e := row.Scan(&item.ID, &item.Output.Kind, &fgID, &fgCode, &fgName, &fgUnit, &rmID, &rmCode, &rmName, &rmUnit, &item.Revision, &item.Status, &item.Notes); e != nil {
		return item, e
	}
	if item.Output.Kind == OutputFG {
		item.Output.ID = fgID
		item.Output.ItemCode = fgCode
		item.Output.Name = fgName
		item.Output.Unit = fgUnit
	} else {
		item.Output.ID = rmID
		item.Output.ItemCode = rmCode
		item.Output.Name = rmName
		item.Output.Unit = rmUnit
	}
	rows, e := tx.Query(ctx, `SELECT r.id,r.code,r.name,u.code,c.usage_qty FROM bom_components c JOIN raw_materials r ON r.tenant_id=c.tenant_id AND r.id=c.raw_material_id JOIN units u ON u.tenant_id=r.tenant_id AND u.id=r.base_unit_id WHERE c.tenant_id=$1 AND c.bom_id=$2 ORDER BY c.sort_position`, tenantID, item.ID)
	if e != nil {
		return item, e
	}
	defer rows.Close()
	for rows.Next() {
		var c Component
		if e = rows.Scan(&c.RawMaterialID, &c.ItemCode, &c.Name, &c.Unit, &c.UsageQty); e != nil {
			return item, e
		}
		item.Components = append(item.Components, c)
	}
	return item, rows.Err()
}
func (s *Store) Create(ctx context.Context, a Actor, input Input) (item BOM, err error) {
	err = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		var revision int
		if input.Output.Kind == OutputFG {
			e := tx.QueryRow(ctx, `SELECT COALESCE(MAX(revision),0)+1 FROM boms WHERE tenant_id=$1 AND finished_good_id=$2`, a.TenantID, input.Output.ID).Scan(&revision)
			if e != nil {
				return e
			}
		} else {
			e := tx.QueryRow(ctx, `SELECT COALESCE(MAX(revision),0)+1 FROM boms WHERE tenant_id=$1 AND raw_material_id=$2`, a.TenantID, input.Output.ID).Scan(&revision)
			if e != nil {
				return e
			}
		}
		var id uuid.UUID
		var e error
		if input.Output.Kind == OutputFG {
			e = tx.QueryRow(ctx, `INSERT INTO boms(tenant_id,output_kind,finished_good_id,revision,notes,created_by_user_id,updated_by_user_id) VALUES($1,'FG',$2,$3,$4,$5,$5) RETURNING id`, a.TenantID, input.Output.ID, revision, input.Notes, a.UserID).Scan(&id)
		} else {
			e = tx.QueryRow(ctx, `INSERT INTO boms(tenant_id,output_kind,raw_material_id,revision,notes,created_by_user_id,updated_by_user_id) VALUES($1,'RAW_MATERIAL',$2,$3,$4,$5,$5) RETURNING id`, a.TenantID, input.Output.ID, revision, input.Notes, a.UserID).Scan(&id)
		}
		if e != nil {
			return e
		}
		for pos, c := range input.Components {
			if _, e = tx.Exec(ctx, `INSERT INTO bom_components(tenant_id,bom_id,raw_material_id,usage_qty,sort_position) VALUES($1,$2,$3,$4,$5)`, a.TenantID, id, c.RawMaterialID, c.UsageQty, pos); e != nil {
				return e
			}
		}
		item.ID = id
		return nil
	})
	if err != nil {
		return item, err
	}
	return s.Get(ctx, a, item.ID)
}
func (s *Store) Activate(ctx context.Context, a Actor, id uuid.UUID) (item BOM, err error) {
	err = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		var kind string
		var fg, rm uuid.UUID
		if e := tx.QueryRow(ctx, `SELECT output_kind,COALESCE(finished_good_id,'00000000-0000-0000-0000-000000000000'),COALESCE(raw_material_id,'00000000-0000-0000-0000-000000000000') FROM boms WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, a.TenantID, id).Scan(&kind, &fg, &rm); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('bom-activation-'||$1::text))`, a.TenantID); e != nil {
			return e
		}
		if kind == OutputFG {
			if _, e := tx.Exec(ctx, `UPDATE boms SET status='REVISED',updated_by_user_id=$3 WHERE tenant_id=$1 AND finished_good_id=$2 AND status='ACTIVE'`, a.TenantID, fg, a.UserID); e != nil {
				return e
			}
		} else {
			if _, e := tx.Exec(ctx, `UPDATE boms SET status='REVISED',updated_by_user_id=$3 WHERE tenant_id=$1 AND raw_material_id=$2 AND status='ACTIVE'`, a.TenantID, rm, a.UserID); e != nil {
				return e
			}
		}
		tag, e := tx.Exec(ctx, `UPDATE boms SET status='ACTIVE',updated_by_user_id=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2 AND status='DRAFT'`, a.TenantID, id, a.UserID)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("only draft BOMs can be activated")
		}
		return nil
	})
	if err != nil {
		return item, err
	}
	return s.Get(ctx, a, id)
}
