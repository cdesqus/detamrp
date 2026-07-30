package masterdata

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"order-stock/backend/internal/database"
)

type SQLStore struct{ db *database.Pool }

func NewSQLStore(db *database.Pool) *SQLStore { return &SQLStore{db: db} }

const unitSelect = `SELECT m.id, m.code, m.name, m.decimal_allowed, m.active,
 m.created_by_user_id, cu.display_name, m.created_at, m.updated_by_user_id, uu.display_name, m.updated_at
 FROM units m JOIN users cu ON cu.tenant_id=m.tenant_id AND cu.id=m.created_by_user_id
 JOIN users uu ON uu.tenant_id=m.tenant_id AND uu.id=m.updated_by_user_id`

func scanUnit(row pgx.Row) (Unit, error) {
	var m Unit
	err := row.Scan(&m.ID, &m.Code, &m.Name, &m.DecimalAllowed, &m.Active, &m.CreatedBy,
		&m.CreatedByName, &m.CreatedAt, &m.UpdatedBy, &m.UpdatedByName, &m.UpdatedAt)
	return m, err
}

func (s *SQLStore) ListUnits(ctx context.Context, actor Actor, q ListQuery) (items []Unit, total int, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		var active any
		if q.Active != nil {
			active = *q.Active
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM units m WHERE m.tenant_id=$1 AND
 ($2='' OR m.code ILIKE '%'||$2||'%' OR m.name ILIKE '%'||$2||'%') AND ($3::boolean IS NULL OR m.active=$3)`, actor.TenantID, q.Search, active).Scan(&total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, unitSelect+` WHERE m.tenant_id=$1 AND
 ($2='' OR m.code ILIKE '%'||$2||'%' OR m.name ILIKE '%'||$2||'%') AND ($3::boolean IS NULL OR m.active=$3)
 ORDER BY m.code LIMIT $4 OFFSET $5`, actor.TenantID, q.Search, active, q.Limit, q.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			m, err := scanUnit(rows)
			if err != nil {
				return err
			}
			items = append(items, m)
		}
		return rows.Err()
	})
	return
}

func (s *SQLStore) GetUnit(ctx context.Context, actor Actor, id uuid.UUID) (item Unit, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		var scanErr error
		item, scanErr = scanUnit(tx.QueryRow(ctx, unitSelect+` WHERE m.tenant_id=$1 AND m.id=$2`, actor.TenantID, id))
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return NotFoundError{Resource: "unit"}
		}
		return scanErr
	})
	return
}

func activeValue(active *bool) bool {
	if active == nil {
		return true
	}
	return *active
}

func (s *SQLStore) CreateUnit(ctx context.Context, actor Actor, in UnitInput) (item Unit, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `INSERT INTO units (tenant_id,code,name,decimal_allowed,active,created_by_user_id,updated_by_user_id)
 VALUES ($1,$2,$3,$4,$5,$6,$6) RETURNING id`, actor.TenantID, in.Code, in.Name, in.DecimalAllowed, activeValue(in.Active), actor.UserID).Scan(&id)
		if err != nil {
			return masterDataWriteError(err, "code")
		}
		item, err = scanUnit(tx.QueryRow(ctx, unitSelect+` WHERE m.tenant_id=$1 AND m.id=$2`, actor.TenantID, id))
		return err
	})
	return
}

func (s *SQLStore) UpdateUnit(ctx context.Context, actor Actor, id uuid.UUID, in UnitInput) (item Unit, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		tag, err := tx.Exec(ctx, `UPDATE units SET code=$3,name=$4,decimal_allowed=$5,active=$6,updated_by_user_id=$7,updated_at=now()
 WHERE tenant_id=$1 AND id=$2`, actor.TenantID, id, in.Code, in.Name, in.DecimalAllowed, activeValue(in.Active), actor.UserID)
		if err != nil {
			return masterDataWriteError(err, "code")
		}
		if tag.RowsAffected() == 0 {
			return NotFoundError{Resource: "unit"}
		}
		item, err = scanUnit(tx.QueryRow(ctx, unitSelect+` WHERE m.tenant_id=$1 AND m.id=$2`, actor.TenantID, id))
		return err
	})
	return
}

func masterDataWriteError(err error, field string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ConflictError{Fields: FieldErrors{field: "Already in use"}}
	}
	return err
}

const supplierSelect = `SELECT s.id,s.code,s.sage_supplier_code,s.name,s.email,s.phone,s.address,s.contact_person,s.currency,s.active,
 s.created_by_user_id,cu.display_name,s.created_at,s.updated_by_user_id,uu.display_name,s.updated_at FROM suppliers s
 JOIN users cu ON cu.tenant_id=s.tenant_id AND cu.id=s.created_by_user_id JOIN users uu ON uu.tenant_id=s.tenant_id AND uu.id=s.updated_by_user_id`

func scanSupplier(row pgx.Row) (Supplier, error) {
	var s Supplier
	err := row.Scan(&s.ID, &s.Code, &s.SageSupplierCode, &s.Name, &s.Email, &s.Phone, &s.Address, &s.ContactPerson, &s.Currency, &s.Active, &s.CreatedBy, &s.CreatedByName, &s.CreatedAt, &s.UpdatedBy, &s.UpdatedByName, &s.UpdatedAt)
	return s, err
}
func (s *SQLStore) ListSuppliers(ctx context.Context, a Actor, q ListQuery) (items []Supplier, total int, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		var active any
		if q.Active != nil {
			active = *q.Active
		}
		filter := ` WHERE s.tenant_id=$1 AND ($2='' OR s.code ILIKE '%'||$2||'%' OR s.name ILIKE '%'||$2||'%' OR s.sage_supplier_code ILIKE '%'||$2||'%') AND ($3::boolean IS NULL OR s.active=$3)`
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM suppliers s`+filter, a.TenantID, q.Search, active).Scan(&total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, supplierSelect+filter+` ORDER BY s.code LIMIT $4 OFFSET $5`, a.TenantID, q.Search, active, q.Limit, q.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item, e := scanSupplier(rows)
			if e != nil {
				return e
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return
}
func (s *SQLStore) GetSupplier(ctx context.Context, a Actor, id uuid.UUID) (item Supplier, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		var e error
		item, e = scanSupplier(tx.QueryRow(ctx, supplierSelect+` WHERE s.tenant_id=$1 AND s.id=$2`, a.TenantID, id))
		if errors.Is(e, pgx.ErrNoRows) {
			return NotFoundError{Resource: "supplier"}
		}
		return e
	})
	return
}
func (s *SQLStore) CreateSupplier(ctx context.Context, a Actor, in SupplierInput) (item Supplier, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		var id uuid.UUID
		e := tx.QueryRow(ctx, `INSERT INTO suppliers(tenant_id,code,sage_supplier_code,name,email,phone,address,contact_person,currency,active,created_by_user_id,updated_by_user_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11) RETURNING id`, a.TenantID, in.Code, in.SageSupplierCode, in.Name, in.Email, in.Phone, in.Address, in.ContactPerson, in.Currency, activeValue(in.Active), a.UserID).Scan(&id)
		if e != nil {
			return supplierWriteError(e)
		}
		item, e = scanSupplier(tx.QueryRow(ctx, supplierSelect+` WHERE s.tenant_id=$1 AND s.id=$2`, a.TenantID, id))
		return e
	})
	return
}
func (s *SQLStore) UpdateSupplier(ctx context.Context, a Actor, id uuid.UUID, in SupplierInput) (item Supplier, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		tag, e := tx.Exec(ctx, `UPDATE suppliers SET code=$3,sage_supplier_code=$4,name=$5,email=$6,phone=$7,address=$8,contact_person=$9,currency=$10,active=$11,updated_by_user_id=$12,updated_at=now() WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, in.Code, in.SageSupplierCode, in.Name, in.Email, in.Phone, in.Address, in.ContactPerson, in.Currency, activeValue(in.Active), a.UserID)
		if e != nil {
			return supplierWriteError(e)
		}
		if tag.RowsAffected() == 0 {
			return NotFoundError{Resource: "supplier"}
		}
		item, e = scanSupplier(tx.QueryRow(ctx, supplierSelect+` WHERE s.tenant_id=$1 AND s.id=$2`, a.TenantID, id))
		return e
	})
	return
}
func supplierWriteError(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) && p.Code == "23505" {
		field := "code"
		if p.ConstraintName == "suppliers_tenant_id_sage_supplier_code_key" {
			field = "sageSupplierCode"
		}
		return ConflictError{Fields: FieldErrors{field: "Already in use"}}
	}
	return err
}

const rawMaterialSelect = `SELECT r.id,r.code,r.sage_item_code,r.name,r.supplier_id,s.name,r.base_unit_id,m.code,
 COALESCE(r.category_id,'00000000-0000-0000-0000-000000000000'),COALESCE(c.code,''),COALESCE(c.name,''),
 COALESCE(r.packing_id,'00000000-0000-0000-0000-000000000000'),COALESCE(p.code,''),COALESCE(p.name,''),
 r.qty_per_kanban,r.minimum_stock,r.standard_unit_price,r.currency,r.description,r.active,
 r.created_by_user_id,cu.display_name,r.created_at,r.updated_by_user_id,uu.display_name,r.updated_at
 FROM raw_materials r
 JOIN suppliers s ON s.tenant_id=r.tenant_id AND s.id=r.supplier_id
 JOIN units m ON m.tenant_id=r.tenant_id AND m.id=r.base_unit_id
 LEFT JOIN categories c ON c.tenant_id=r.tenant_id AND c.id=r.category_id
 LEFT JOIN packings p ON p.tenant_id=r.tenant_id AND p.id=r.packing_id
 JOIN users cu ON cu.tenant_id=r.tenant_id AND cu.id=r.created_by_user_id
 JOIN users uu ON uu.tenant_id=r.tenant_id AND uu.id=r.updated_by_user_id`

func scanRawMaterial(row pgx.Row) (RawMaterial, error) {
	var r RawMaterial
	err := row.Scan(&r.ID, &r.Code, &r.SageItemCode, &r.Name, &r.SupplierID, &r.SupplierName, &r.BaseUnitID, &r.BaseUnitCode,
		&r.CategoryID, &r.CategoryCode, &r.CategoryName, &r.PackingID, &r.PackingCode, &r.PackingName,
		&r.QtyPerKanban, &r.MinimumStock, &r.StandardUnitPrice, &r.Currency, &r.Description, &r.Active,
		&r.CreatedBy, &r.CreatedByName, &r.CreatedAt, &r.UpdatedBy, &r.UpdatedByName, &r.UpdatedAt)
	return r, err
}
func (s *SQLStore) ListRawMaterials(ctx context.Context, a Actor, q ListQuery) (items []RawMaterial, total int, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		var active any
		if q.Active != nil {
			active = *q.Active
		}
		filter := ` WHERE r.tenant_id=$1 AND ($2='' OR r.code ILIKE '%'||$2||'%' OR r.name ILIKE '%'||$2||'%' OR r.sage_item_code ILIKE '%'||$2||'%') AND ($3::boolean IS NULL OR r.active=$3) AND ($4::uuid='00000000-0000-0000-0000-000000000000' OR r.supplier_id=$4)`
		if e := tx.QueryRow(ctx, `SELECT count(*) FROM raw_materials r`+filter, a.TenantID, q.Search, active, q.SupplierID).Scan(&total); e != nil {
			return e
		}
		rows, e := tx.Query(ctx, rawMaterialSelect+filter+` ORDER BY r.code LIMIT $5 OFFSET $6`, a.TenantID, q.Search, active, q.SupplierID, q.Limit, q.Offset)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			item, e := scanRawMaterial(rows)
			if e != nil {
				return e
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return
}
func (s *SQLStore) GetRawMaterial(ctx context.Context, a Actor, id uuid.UUID) (item RawMaterial, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		var e error
		item, e = scanRawMaterial(tx.QueryRow(ctx, rawMaterialSelect+` WHERE r.tenant_id=$1 AND r.id=$2`, a.TenantID, id))
		if errors.Is(e, pgx.ErrNoRows) {
			return NotFoundError{Resource: "raw material"}
		}
		return e
	})
	return
}
func (s *SQLStore) CreateRawMaterial(ctx context.Context, a Actor, in RawMaterialInput) (item RawMaterial, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		currency, referenceErr := activeRawMaterialReferences(ctx, tx, a.TenantID, in)
		if referenceErr != nil {
			return referenceErr
		}
		var id uuid.UUID
		e := tx.QueryRow(ctx, `INSERT INTO raw_materials(tenant_id,code,sage_item_code,name,supplier_id,base_unit_id,category_id,packing_id,
 qty_per_kanban,minimum_stock,standard_unit_price,currency,description,active,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$15) RETURNING id`,
			a.TenantID, in.Code, in.SageItemCode, in.Name, in.SupplierID, in.BaseUnitID, in.CategoryID, in.PackingID,
			in.QtyPerKanban, in.MinimumStock, in.StandardUnitPrice, currency, in.Description, activeValue(in.Active), a.UserID).Scan(&id)
		if e != nil {
			return rawMaterialWriteError(e)
		}
		item, e = scanRawMaterial(tx.QueryRow(ctx, rawMaterialSelect+` WHERE r.tenant_id=$1 AND r.id=$2`, a.TenantID, id))
		return e
	})
	return
}
func (s *SQLStore) UpdateRawMaterial(ctx context.Context, a Actor, id uuid.UUID, in RawMaterialInput) (item RawMaterial, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		currency, referenceErr := activeRawMaterialReferences(ctx, tx, a.TenantID, in)
		if referenceErr != nil {
			return referenceErr
		}
		tag, e := tx.Exec(ctx, `UPDATE raw_materials SET code=$3,sage_item_code=$4,name=$5,supplier_id=$6,base_unit_id=$7,
 category_id=$8,packing_id=$9,qty_per_kanban=$10,minimum_stock=$11,standard_unit_price=$12,currency=$13,description=$14,
 active=$15,updated_by_user_id=$16,updated_at=now() WHERE tenant_id=$1 AND id=$2`,
			a.TenantID, id, in.Code, in.SageItemCode, in.Name, in.SupplierID, in.BaseUnitID, in.CategoryID, in.PackingID,
			in.QtyPerKanban, in.MinimumStock, in.StandardUnitPrice, currency, in.Description, activeValue(in.Active), a.UserID)
		if e != nil {
			return rawMaterialWriteError(e)
		}
		if tag.RowsAffected() == 0 {
			return NotFoundError{Resource: "raw material"}
		}
		item, e = scanRawMaterial(tx.QueryRow(ctx, rawMaterialSelect+` WHERE r.tenant_id=$1 AND r.id=$2`, a.TenantID, id))
		return e
	})
	return
}

func activeRawMaterialReferences(ctx context.Context, tx database.TenantTx, tenantID uuid.UUID, input RawMaterialInput) (string, error) {
	var currency string
	var unitActive, categoryActive, packingActive bool
	err := tx.QueryRow(ctx, `SELECT
 COALESCE((SELECT s.currency FROM suppliers s WHERE s.tenant_id=$1 AND s.id=$2 AND s.active=true),''),
 EXISTS(SELECT 1 FROM units u WHERE u.tenant_id=$1 AND u.id=$3 AND u.active=true),
 EXISTS(SELECT 1 FROM categories c WHERE c.tenant_id=$1 AND c.id=$4 AND c.active=true),
 EXISTS(SELECT 1 FROM packings p WHERE p.tenant_id=$1 AND p.id=$5 AND p.active=true)`,
		tenantID, input.SupplierID, input.BaseUnitID, input.CategoryID, input.PackingID).
		Scan(&currency, &unitActive, &categoryActive, &packingActive)
	if err != nil {
		return "", err
	}
	switch {
	case currency == "":
		return "", ConflictError{Fields: FieldErrors{"supplierId": "Select an active supplier"}}
	case !unitActive:
		return "", ConflictError{Fields: FieldErrors{"baseUnitId": "Select an active base unit"}}
	case !categoryActive:
		return "", ConflictError{Fields: FieldErrors{"categoryId": "Select an active category"}}
	case !packingActive:
		return "", ConflictError{Fields: FieldErrors{"packingId": "Select an active packing"}}
	default:
		return currency, nil
	}
}
func rawMaterialWriteError(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) {
		if p.Code == "23505" {
			field := "code"
			if p.ConstraintName == "raw_materials_tenant_id_sage_item_code_key" {
				field = "sageItemCode"
			}
			return ConflictError{Fields: FieldErrors{field: "Already in use"}}
		}
		if p.Code == "23503" {
			field := "baseUnitId"
			return ConflictError{Fields: FieldErrors{field: "Select a valid active base unit"}}
		}
	}
	return err
}
