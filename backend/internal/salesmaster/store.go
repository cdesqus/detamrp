package salesmaster

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"order-stock/backend/internal/database"
	"order-stock/backend/internal/rbac"
)

type Store struct{ db *database.Pool }

func NewStore(db *database.Pool) *Store { return &Store{db: db} }

const customerSelect = `SELECT c.id,c.code,c.name,c.address,c.contact,c.email,c.phone,c.active,
 c.created_by_user_id,cu.display_name,c.created_at,c.updated_by_user_id,uu.display_name,c.updated_at
 FROM customers c
 JOIN users cu ON cu.tenant_id=c.tenant_id AND cu.id=c.created_by_user_id
 JOIN users uu ON uu.tenant_id=c.tenant_id AND uu.id=c.updated_by_user_id`

func scanCustomer(row pgx.Row) (Customer, error) {
	var item Customer
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.Address, &item.Contact, &item.Email, &item.Phone, &item.Active,
		&item.CreatedBy, &item.CreatedByName, &item.CreatedAt, &item.UpdatedBy, &item.UpdatedByName, &item.UpdatedAt)
	return item, err
}

func (s *Store) ListCustomers(ctx context.Context, actor Actor, query ListQuery) (items []Customer, total int, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		filter := ` WHERE c.tenant_id=$1 AND ($2='' OR c.code ILIKE '%'||$2||'%' OR c.name ILIKE '%'||$2||'%' OR c.contact ILIKE '%'||$2||'%' OR c.email ILIKE '%'||$2||'%')
 AND ($3::boolean IS NULL OR c.active=$3)`
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM customers c`+filter, actor.TenantID, query.Search, nullableActive(query.Active)).Scan(&total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, customerSelect+filter+` ORDER BY c.code LIMIT $4 OFFSET $5`, actor.TenantID, query.Search, nullableActive(query.Active), query.Limit, query.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item, err := scanCustomer(rows)
			if err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return
}

func (s *Store) GetCustomer(ctx context.Context, actor Actor, id uuid.UUID) (item Customer, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		var scanErr error
		item, scanErr = scanCustomer(tx.QueryRow(ctx, customerSelect+` WHERE c.tenant_id=$1 AND c.id=$2`, actor.TenantID, id))
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return NotFoundError{Resource: "customer"}
		}
		return scanErr
	})
	return
}

func (s *Store) CreateCustomer(ctx context.Context, actor Actor, input CustomerInput) (item Customer, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		var id uuid.UUID
		if err := tx.QueryRow(ctx, `INSERT INTO customers(tenant_id,code,name,address,contact,email,phone,active,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9) RETURNING id`, actor.TenantID, input.Code, input.Name, input.Address, input.Contact, input.Email, input.Phone, activeValue(input.Active), actor.UserID).Scan(&id); err != nil {
			return salesMasterWriteError(err, "code")
		}
		item, err = scanCustomer(tx.QueryRow(ctx, customerSelect+` WHERE c.tenant_id=$1 AND c.id=$2`, actor.TenantID, id))
		return err
	})
	return
}

func (s *Store) UpdateCustomer(ctx context.Context, actor Actor, id uuid.UUID, input CustomerInput) (item Customer, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		tag, err := tx.Exec(ctx, `UPDATE customers SET code=$3,name=$4,address=$5,contact=$6,email=$7,phone=$8,active=$9,updated_by_user_id=$10,updated_at=now()
 WHERE tenant_id=$1 AND id=$2`, actor.TenantID, id, input.Code, input.Name, input.Address, input.Contact, input.Email, input.Phone, activeValue(input.Active), actor.UserID)
		if err != nil {
			return salesMasterWriteError(err, "code")
		}
		if tag.RowsAffected() == 0 {
			return NotFoundError{Resource: "customer"}
		}
		item, err = scanCustomer(tx.QueryRow(ctx, customerSelect+` WHERE c.tenant_id=$1 AND c.id=$2`, actor.TenantID, id))
		return err
	})
	return
}

const finishedGoodSelect = `SELECT f.id,f.item_code,f.name,f.base_unit_id,u.code,u.name,f.sales_price,f.currency,f.price_version,f.active,
 f.created_by_user_id,cu.display_name,f.created_at,f.updated_by_user_id,uu.display_name,f.updated_at
 FROM finished_goods f
 JOIN units u ON u.tenant_id=f.tenant_id AND u.id=f.base_unit_id
 JOIN users cu ON cu.tenant_id=f.tenant_id AND cu.id=f.created_by_user_id
 JOIN users uu ON uu.tenant_id=f.tenant_id AND uu.id=f.updated_by_user_id`

func scanFinishedGood(row pgx.Row) (FinishedGood, error) {
	var item FinishedGood
	err := row.Scan(&item.ID, &item.ItemCode, &item.Name, &item.BaseUnitID, &item.BaseUnitCode, &item.BaseUnitName,
		&item.SalesPrice, &item.Currency, &item.PriceVersion, &item.Active,
		&item.CreatedBy, &item.CreatedByName, &item.CreatedAt, &item.UpdatedBy, &item.UpdatedByName, &item.UpdatedAt)
	return item, err
}

func (s *Store) ListFinishedGoods(ctx context.Context, actor Actor, query ListQuery) (items []FinishedGood, total int, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		filter := ` WHERE f.tenant_id=$1 AND ($2='' OR f.item_code ILIKE '%'||$2||'%' OR f.name ILIKE '%'||$2||'%')
 AND ($3::boolean IS NULL OR f.active=$3)`
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM finished_goods f`+filter, actor.TenantID, query.Search, nullableActive(query.Active)).Scan(&total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, finishedGoodSelect+filter+` ORDER BY f.item_code LIMIT $4 OFFSET $5`, actor.TenantID, query.Search, nullableActive(query.Active), query.Limit, query.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item, err := scanFinishedGood(rows)
			if err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return
}

func (s *Store) GetFinishedGood(ctx context.Context, actor Actor, id uuid.UUID) (item FinishedGood, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		var scanErr error
		item, scanErr = scanFinishedGood(tx.QueryRow(ctx, finishedGoodSelect+` WHERE f.tenant_id=$1 AND f.id=$2`, actor.TenantID, id))
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return NotFoundError{Resource: "finished good"}
		}
		return scanErr
	})
	return
}

func (s *Store) CreateFinishedGood(ctx context.Context, actor Actor, input FinishedGoodInput) (item FinishedGood, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		var id uuid.UUID
		if err := tx.QueryRow(ctx, `INSERT INTO finished_goods(tenant_id,item_code,name,base_unit_id,sales_price,currency,active,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8) RETURNING id`, actor.TenantID, input.ItemCode, input.Name, input.BaseUnitID, input.SalesPrice, input.Currency, activeValue(input.Active), actor.UserID).Scan(&id); err != nil {
			return finishedGoodWriteError(err)
		}
		item, err = scanFinishedGood(tx.QueryRow(ctx, finishedGoodSelect+` WHERE f.tenant_id=$1 AND f.id=$2`, actor.TenantID, id))
		return err
	})
	return
}

func (s *Store) UpdateFinishedGood(ctx context.Context, actor Actor, id uuid.UUID, input FinishedGoodInput) (item FinishedGood, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		var tag pgconn.CommandTag
		var err error
		if rbac.Allows(actor.Permissions, "fg.price.manage") {
			tag, err = tx.Exec(ctx, `UPDATE finished_goods SET item_code=$3,name=$4,base_unit_id=$5,
 price_version=CASE WHEN sales_price IS DISTINCT FROM $6 OR currency IS DISTINCT FROM $7 THEN price_version+1 ELSE price_version END,
 sales_price=$6,currency=$7,active=$8,updated_by_user_id=$9,updated_at=now() WHERE tenant_id=$1 AND id=$2`,
				actor.TenantID, id, input.ItemCode, input.Name, input.BaseUnitID, input.SalesPrice, input.Currency, activeValue(input.Active), actor.UserID)
		} else {
			tag, err = tx.Exec(ctx, `UPDATE finished_goods SET item_code=$3,name=$4,base_unit_id=$5,
 active=$6,updated_by_user_id=$7,updated_at=now() WHERE tenant_id=$1 AND id=$2`,
				actor.TenantID, id, input.ItemCode, input.Name, input.BaseUnitID, activeValue(input.Active), actor.UserID)
		}
		if err != nil {
			return finishedGoodWriteError(err)
		}
		if tag.RowsAffected() == 0 {
			return NotFoundError{Resource: "finished good"}
		}
		item, err = scanFinishedGood(tx.QueryRow(ctx, finishedGoodSelect+` WHERE f.tenant_id=$1 AND f.id=$2`, actor.TenantID, id))
		return err
	})
	return
}

func tenantContext(actor Actor) database.TenantContext {
	return database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}
}

func nullableActive(active *bool) any {
	if active == nil {
		return nil
	}
	return *active
}

func salesMasterWriteError(err error, duplicateField string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ConflictError{Fields: FieldErrors{duplicateField: "Code is already in use"}}
	}
	return err
}

func finishedGoodWriteError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	if pgErr.Code == "23505" {
		return ConflictError{Fields: FieldErrors{"itemCode": "Item Code is already in use"}}
	}
	if pgErr.Code == "23503" && pgErr.ConstraintName == "finished_goods_tenant_id_base_unit_id_fkey" {
		return ConflictError{Fields: FieldErrors{"baseUnitId": "Select an existing Base Unit"}}
	}
	return err
}
