package inventory

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"order-stock/backend/internal/database"
)

type Store struct {
	db *database.Pool
}

func NewStore(db *database.Pool) *Store {
	return &Store{db: db}
}

func tenant(actor Actor) database.TenantContext {
	return database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}
}

func (s *Store) ListStock(ctx context.Context, actor Actor, filters Filters) (StockResponse, error) {
	response := StockResponse{Items: []StockItem{}}
	var all []StockItem
	err := database.WithTenant(ctx, s.db, tenant(actor), func(tx database.TenantTx) error {
		rows, err := tx.Query(ctx, `
SELECT rm.id, rm.code, rm.name, s.id, s.name,
       COUNT(kl.id), COALESCE(SUM(kl.quantity), 0),
       m.code, rm.minimum_stock
FROM raw_materials rm
JOIN suppliers s ON s.tenant_id = rm.tenant_id AND s.id = rm.supplier_id
JOIN units m ON m.tenant_id = rm.tenant_id AND m.id = rm.base_unit_id
LEFT JOIN purchase_order_lines pol
  ON pol.tenant_id = rm.tenant_id AND pol.raw_material_id = rm.id
LEFT JOIN kanban_lots kl
  ON kl.tenant_id = pol.tenant_id
 AND kl.purchase_order_line_id = pol.id
 AND kl.status = 'IN_STOCK'
WHERE rm.tenant_id = $1 AND rm.active = true
GROUP BY rm.id, rm.code, rm.name, s.id, s.name, m.code, rm.minimum_stock
ORDER BY rm.code`, actor.TenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item StockItem
			if err := rows.Scan(
				&item.RawMaterialID,
				&item.ItemCode,
				&item.RawMaterialName,
				&item.SupplierID,
				&item.SupplierName,
				&item.AvailableKanban,
				&item.StockQuantity,
				&item.BaseUnitCode,
				&item.MinimumStock,
			); err != nil {
				return err
			}
			item.StockStatus = StockStatus(item.StockQuantity, item.MinimumStock)
			all = append(all, item)
		}
		return rows.Err()
	})
	if err != nil {
		return StockResponse{}, err
	}
	response.Summary = summarizeStock(all)
	response.Items = filterStockItems(all, filters)
	return response, nil
}

func (s *Store) ListKanbans(ctx context.Context, actor Actor, rawMaterialID uuid.UUID) (KanbanResponse, error) {
	response := KanbanResponse{RawMaterialID: rawMaterialID, Kanbans: []KanbanItem{}}
	err := database.WithTenant(ctx, s.db, tenant(actor), func(tx database.TenantTx) error {
		err := tx.QueryRow(ctx, `
SELECT code, name
FROM raw_materials
WHERE tenant_id = $1 AND id = $2 AND active = true`,
			actor.TenantID, rawMaterialID,
		).Scan(&response.ItemCode, &response.RawMaterialName)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		rows, err := tx.Query(ctx, `
SELECT kl.id, kl.kanban_id, dn.delivery_note_number, po.po_number,
       kl.quantity, pol.base_unit_code_snapshot, r.receiving_date
FROM kanban_lots kl
JOIN purchase_order_lines pol
  ON pol.tenant_id = kl.tenant_id AND pol.id = kl.purchase_order_line_id
JOIN delivery_note_lines dnl
  ON dnl.tenant_id = kl.tenant_id AND dnl.id = kl.delivery_note_line_id
JOIN delivery_notes dn
  ON dn.tenant_id = dnl.tenant_id AND dn.id = dnl.delivery_note_id
JOIN purchase_orders po
  ON po.tenant_id = dn.tenant_id AND po.id = dn.purchase_order_id
JOIN receiving_kanban_lots rkl
  ON rkl.tenant_id = kl.tenant_id AND rkl.kanban_lot_id = kl.id
JOIN receivings r
  ON r.tenant_id = rkl.tenant_id AND r.id = rkl.receiving_id
WHERE kl.tenant_id = $1
  AND pol.raw_material_id = $2
  AND kl.status = 'IN_STOCK'
ORDER BY kl.kanban_id`, actor.TenantID, rawMaterialID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item KanbanItem
			if err := rows.Scan(
				&item.KanbanLotID,
				&item.KanbanID,
				&item.DeliveryNoteNumber,
				&item.PONumber,
				&item.Quantity,
				&item.BaseUnitCode,
				&item.ReceivedDate,
			); err != nil {
				return err
			}
			response.Kanbans = append(response.Kanbans, item)
		}
		return rows.Err()
	})
	if err != nil {
		return KanbanResponse{}, err
	}
	return response, nil
}

func summarizeStock(items []StockItem) Summary {
	summary := Summary{TotalRawMaterials: len(items)}
	for _, item := range items {
		summary.TotalInStockKanban += item.AvailableKanban
		switch item.StockStatus {
		case "LOW_STOCK":
			summary.LowStockMaterials++
		case "OUT_OF_STOCK":
			summary.OutOfStockMaterials++
		}
	}
	return summary
}

func filterStockItems(items []StockItem, filters Filters) []StockItem {
	result := make([]StockItem, 0, len(items))
	search := strings.ToLower(strings.TrimSpace(filters.Search))
	for _, item := range items {
		if search != "" &&
			!strings.Contains(strings.ToLower(item.ItemCode), search) &&
			!strings.Contains(strings.ToLower(item.RawMaterialName), search) {
			continue
		}
		if filters.SupplierID != nil && item.SupplierID != *filters.SupplierID {
			continue
		}
		if filters.Status != "" && item.StockStatus != filters.Status {
			continue
		}
		result = append(result, item)
	}
	return result
}
