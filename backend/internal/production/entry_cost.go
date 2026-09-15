package production

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/database"
)

func validateEntryCosts(material, process decimal.Decimal) error {
	if material.GreaterThanOrEqual(decimal.New(1, 14)) || process.GreaterThanOrEqual(decimal.New(1, 14)) {
		return invalid("Entry cost is too large; split the entry into smaller quantities")
	}
	return nil
}
func entryCosts(ctx context.Context, tx database.TenantTx, a Actor, o Order, index int, i EntryInput, old *Entry) (materials []MaterialUsage, total, rate decimal.Decimal, err error) {
	materials = []MaterialUsage{}
	rate = o.Operations[index].Rate
	if old != nil {
		rate = old.ProcessRate
	} else {
		var raw []byte
		var currency string
		e := tx.QueryRow(ctx, `SELECT r.steps,r.currency FROM production_routings r JOIN production_orders o ON o.tenant_id=r.tenant_id AND (r.finished_good_id=o.finished_good_id OR r.raw_material_id=o.raw_material_id) WHERE r.tenant_id=$1 AND o.id=$2 AND r.active FOR SHARE OF r`, a.TenantID, o.ID).Scan(&raw, &currency)
		if e != nil {
			return nil, total, rate, invalid("An active routing is required to snapshot the current process rate")
		}
		if currency != o.Currency {
			return nil, total, rate, invalid("Active routing currency differs from the order")
		}
		var steps []RoutingStep
		if e = json.Unmarshal(raw, &steps); e != nil {
			return nil, total, rate, e
		}
		found := false
		for _, step := range steps {
			if step.Code == o.Operations[index].Code {
				rate = step.Rate
				found = true
				break
			}
		}
		if !found {
			return nil, total, rate, invalid("The active routing has no rate for operation %s", o.Operations[index].Code)
		}
	}
	if index > 0 {
		if len(i.Materials) > 0 {
			return nil, total, rate, invalid("Material usage is recorded at the first operation only")
		}
		return
	}
	expected := map[uuid.UUID]MaterialSnapshot{}
	for _, m := range o.Materials {
		expected[m.ID] = m
	}
	if len(expected) == 0 {
		return nil, total, rate, invalid("Order has no material snapshot; correct its setup before recording production")
	}
	if len(i.Materials) != len(expected) {
		return nil, total, rate, invalid("Record actual usage for every BOM material")
	}
	previous := map[uuid.UUID]MaterialUsage{}
	if old != nil {
		for _, m := range old.Materials {
			previous[m.MaterialID] = m
		}
	}
	for _, input := range i.Materials {
		snapshot, ok := expected[input.MaterialID]
		if !ok {
			return nil, total, rate, invalid("Material does not belong to the order BOM")
		}
		m := MaterialUsage{MaterialID: input.MaterialID, Quantity: input.Quantity, PartNumber: snapshot.PartNumber, PartName: snapshot.PartName, UnitCode: snapshot.UnitCode}
		if prior, ok := previous[m.MaterialID]; ok {
			m.UnitPrice = prior.UnitPrice
			m.PartNumber = prior.PartNumber
			m.PartName = prior.PartName
			m.UnitCode = prior.UnitCode
		} else {
			var currency string
			var active bool
			if e := tx.QueryRow(ctx, `SELECT standard_unit_price,trim(currency),active FROM raw_materials WHERE tenant_id=$1 AND id=$2 FOR SHARE`, a.TenantID, m.MaterialID).Scan(&m.UnitPrice, &currency, &active); e != nil {
				return nil, total, rate, e
			}
			if !active || currency != o.Currency {
				return nil, total, rate, invalid("Materials must be active and use the order currency")
			}
		}
		m.Cost = m.Quantity.Mul(m.UnitPrice).Round(6)
		total = total.Add(m.Cost)
		materials = append(materials, m)
	}
	return
}
