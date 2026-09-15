package production

import (
	"context"
	"github.com/google/uuid"
	"order-stock/backend/internal/database"
)

type PartOption struct {
	ID         uuid.UUID `json:"id"`
	Kind       string    `json:"kind"`
	PartNumber string    `json:"partNumber"`
	PartName   string    `json:"partName"`
	UnitCode   string    `json:"unitCode"`
}
type PlantOption struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
type PlanOptions struct {
	Parts  []PartOption  `json:"parts"`
	Plants []PlantOption `json:"plants"`
}

func (s *Store) Options(ctx context.Context, a Actor) (result PlanOptions, err error) {
	result = PlanOptions{Parts: []PartOption{}, Plants: []PlantOption{}}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT id,name FROM plants WHERE tenant_id=$1 AND active ORDER BY name`, a.TenantID)
		if e != nil {
			return e
		}
		for rows.Next() {
			var p PlantOption
			if e = rows.Scan(&p.ID, &p.Name); e != nil {
				rows.Close()
				return e
			}
			result.Plants = append(result.Plants, p)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		rows, e = tx.Query(ctx, `SELECT f.id,'FG',f.item_code,f.name,u.code FROM finished_goods f JOIN units u ON u.tenant_id=f.tenant_id AND u.id=f.base_unit_id WHERE f.tenant_id=$1 AND f.active AND u.active UNION ALL SELECT r.id,'RAW_MATERIAL',r.code,r.name,u.code FROM raw_materials r JOIN units u ON u.tenant_id=r.tenant_id AND u.id=r.base_unit_id WHERE r.tenant_id=$1 AND r.active AND u.active ORDER BY 3`, a.TenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var p PartOption
			if e = rows.Scan(&p.ID, &p.Kind, &p.PartNumber, &p.PartName, &p.UnitCode); e != nil {
				return e
			}
			result.Parts = append(result.Parts, p)
		}
		return rows.Err()
	})
	return
}
