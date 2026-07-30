package masterdata

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"order-stock/backend/internal/database"
)

const categorySelect = `SELECT c.id,c.code,c.name,c.description,c.active,
 c.created_by_user_id,cu.display_name,c.created_at,c.updated_by_user_id,uu.display_name,c.updated_at
 FROM categories c
 JOIN users cu ON cu.tenant_id=c.tenant_id AND cu.id=c.created_by_user_id
 JOIN users uu ON uu.tenant_id=c.tenant_id AND uu.id=c.updated_by_user_id`

func scanCategory(row pgx.Row) (Category, error) {
	var item Category
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Active,
		&item.CreatedBy, &item.CreatedByName, &item.CreatedAt, &item.UpdatedBy, &item.UpdatedByName, &item.UpdatedAt)
	return item, err
}

func (s *SQLStore) ListCategories(ctx context.Context, actor Actor, query ListQuery) (items []Category, total int, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		active := nullableActive(query.Active)
		filter := ` WHERE c.tenant_id=$1 AND ($2='' OR c.code ILIKE '%'||$2||'%' OR c.name ILIKE '%'||$2||'%')
 AND ($3::boolean IS NULL OR c.active=$3)`
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM categories c`+filter, actor.TenantID, query.Search, active).Scan(&total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, categorySelect+filter+` ORDER BY c.code LIMIT $4 OFFSET $5`,
			actor.TenantID, query.Search, active, query.Limit, query.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item, err := scanCategory(rows)
			if err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return
}

func (s *SQLStore) GetCategory(ctx context.Context, actor Actor, id uuid.UUID) (item Category, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		var scanErr error
		item, scanErr = scanCategory(tx.QueryRow(ctx, categorySelect+` WHERE c.tenant_id=$1 AND c.id=$2`, actor.TenantID, id))
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return NotFoundError{Resource: "category"}
		}
		return scanErr
	})
	return
}

func (s *SQLStore) CreateCategory(ctx context.Context, actor Actor, input CategoryInput) (item Category, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `INSERT INTO categories
 (tenant_id,code,name,description,active,created_by_user_id,updated_by_user_id)
 VALUES ($1,$2,$3,$4,$5,$6,$6) RETURNING id`,
			actor.TenantID, input.Code, input.Name, input.Description, activeValue(input.Active), actor.UserID).Scan(&id)
		if err != nil {
			return masterDataWriteError(err, "code")
		}
		item, err = scanCategory(tx.QueryRow(ctx, categorySelect+` WHERE c.tenant_id=$1 AND c.id=$2`, actor.TenantID, id))
		return err
	})
	return
}

func (s *SQLStore) UpdateCategory(ctx context.Context, actor Actor, id uuid.UUID, input CategoryInput) (item Category, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		tag, err := tx.Exec(ctx, `UPDATE categories SET code=$3,name=$4,description=$5,active=$6,
 updated_by_user_id=$7,updated_at=now() WHERE tenant_id=$1 AND id=$2`,
			actor.TenantID, id, input.Code, input.Name, input.Description, activeValue(input.Active), actor.UserID)
		if err != nil {
			return masterDataWriteError(err, "code")
		}
		if tag.RowsAffected() == 0 {
			return NotFoundError{Resource: "category"}
		}
		item, err = scanCategory(tx.QueryRow(ctx, categorySelect+` WHERE c.tenant_id=$1 AND c.id=$2`, actor.TenantID, id))
		return err
	})
	return
}

const packingSelect = `SELECT p.id,p.code,p.name,p.description,p.active,
 p.created_by_user_id,cu.display_name,p.created_at,p.updated_by_user_id,uu.display_name,p.updated_at
 FROM packings p
 JOIN users cu ON cu.tenant_id=p.tenant_id AND cu.id=p.created_by_user_id
 JOIN users uu ON uu.tenant_id=p.tenant_id AND uu.id=p.updated_by_user_id`

func scanPacking(row pgx.Row) (Packing, error) {
	var item Packing
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Active,
		&item.CreatedBy, &item.CreatedByName, &item.CreatedAt, &item.UpdatedBy, &item.UpdatedByName, &item.UpdatedAt)
	return item, err
}

func (s *SQLStore) ListPackings(ctx context.Context, actor Actor, query ListQuery) (items []Packing, total int, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		active := nullableActive(query.Active)
		filter := ` WHERE p.tenant_id=$1 AND ($2='' OR p.code ILIKE '%'||$2||'%' OR p.name ILIKE '%'||$2||'%')
 AND ($3::boolean IS NULL OR p.active=$3)`
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM packings p`+filter, actor.TenantID, query.Search, active).Scan(&total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, packingSelect+filter+` ORDER BY p.code LIMIT $4 OFFSET $5`,
			actor.TenantID, query.Search, active, query.Limit, query.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item, err := scanPacking(rows)
			if err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return
}

func (s *SQLStore) GetPacking(ctx context.Context, actor Actor, id uuid.UUID) (item Packing, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		var scanErr error
		item, scanErr = scanPacking(tx.QueryRow(ctx, packingSelect+` WHERE p.tenant_id=$1 AND p.id=$2`, actor.TenantID, id))
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return NotFoundError{Resource: "packing"}
		}
		return scanErr
	})
	return
}

func (s *SQLStore) CreatePacking(ctx context.Context, actor Actor, input PackingInput) (item Packing, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `INSERT INTO packings
 (tenant_id,code,name,description,active,created_by_user_id,updated_by_user_id)
 VALUES ($1,$2,$3,$4,$5,$6,$6) RETURNING id`,
			actor.TenantID, input.Code, input.Name, input.Description, activeValue(input.Active), actor.UserID).Scan(&id)
		if err != nil {
			return masterDataWriteError(err, "code")
		}
		item, err = scanPacking(tx.QueryRow(ctx, packingSelect+` WHERE p.tenant_id=$1 AND p.id=$2`, actor.TenantID, id))
		return err
	})
	return
}

func (s *SQLStore) UpdatePacking(ctx context.Context, actor Actor, id uuid.UUID, input PackingInput) (item Packing, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		tag, err := tx.Exec(ctx, `UPDATE packings SET code=$3,name=$4,description=$5,active=$6,
 updated_by_user_id=$7,updated_at=now() WHERE tenant_id=$1 AND id=$2`,
			actor.TenantID, id, input.Code, input.Name, input.Description, activeValue(input.Active), actor.UserID)
		if err != nil {
			return masterDataWriteError(err, "code")
		}
		if tag.RowsAffected() == 0 {
			return NotFoundError{Resource: "packing"}
		}
		item, err = scanPacking(tx.QueryRow(ctx, packingSelect+` WHERE p.tenant_id=$1 AND p.id=$2`, actor.TenantID, id))
		return err
	})
	return
}

const plantSelect = `SELECT p.id,p.code,p.name,p.address,p.active,
 p.created_by_user_id,cu.display_name,p.created_at,p.updated_by_user_id,uu.display_name,p.updated_at
 FROM plants p
 JOIN users cu ON cu.tenant_id=p.tenant_id AND cu.id=p.created_by_user_id
 JOIN users uu ON uu.tenant_id=p.tenant_id AND uu.id=p.updated_by_user_id`

func scanPlant(row pgx.Row) (Plant, error) {
	var item Plant
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.Address, &item.Active,
		&item.CreatedBy, &item.CreatedByName, &item.CreatedAt, &item.UpdatedBy, &item.UpdatedByName, &item.UpdatedAt)
	return item, err
}

func (s *SQLStore) ListPlants(ctx context.Context, actor Actor, query ListQuery) (items []Plant, total int, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		active := nullableActive(query.Active)
		filter := ` WHERE p.tenant_id=$1 AND ($2='' OR p.code ILIKE '%'||$2||'%' OR p.name ILIKE '%'||$2||'%')
 AND ($3::boolean IS NULL OR p.active=$3)`
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM plants p`+filter, actor.TenantID, query.Search, active).Scan(&total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, plantSelect+filter+` ORDER BY p.code LIMIT $4 OFFSET $5`,
			actor.TenantID, query.Search, active, query.Limit, query.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item, err := scanPlant(rows)
			if err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return
}

func (s *SQLStore) GetPlant(ctx context.Context, actor Actor, id uuid.UUID) (item Plant, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		var scanErr error
		item, scanErr = scanPlant(tx.QueryRow(ctx, plantSelect+` WHERE p.tenant_id=$1 AND p.id=$2`, actor.TenantID, id))
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return NotFoundError{Resource: "plant"}
		}
		return scanErr
	})
	return
}

func (s *SQLStore) CreatePlant(ctx context.Context, actor Actor, input PlantInput) (item Plant, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `INSERT INTO plants
 (tenant_id,code,name,address,active,created_by_user_id,updated_by_user_id)
 VALUES ($1,$2,$3,$4,$5,$6,$6) RETURNING id`,
			actor.TenantID, input.Code, input.Name, input.Address, activeValue(input.Active), actor.UserID).Scan(&id)
		if err != nil {
			return masterDataWriteError(err, "code")
		}
		item, err = scanPlant(tx.QueryRow(ctx, plantSelect+` WHERE p.tenant_id=$1 AND p.id=$2`, actor.TenantID, id))
		return err
	})
	return
}

func (s *SQLStore) UpdatePlant(ctx context.Context, actor Actor, id uuid.UUID, input PlantInput) (item Plant, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		tag, err := tx.Exec(ctx, `UPDATE plants SET code=$3,name=$4,address=$5,active=$6,
 updated_by_user_id=$7,updated_at=now() WHERE tenant_id=$1 AND id=$2`,
			actor.TenantID, id, input.Code, input.Name, input.Address, activeValue(input.Active), actor.UserID)
		if err != nil {
			return masterDataWriteError(err, "code")
		}
		if tag.RowsAffected() == 0 {
			return NotFoundError{Resource: "plant"}
		}
		item, err = scanPlant(tx.QueryRow(ctx, plantSelect+` WHERE p.tenant_id=$1 AND p.id=$2`, actor.TenantID, id))
		return err
	})
	return
}

func nullableActive(active *bool) any {
	if active == nil {
		return nil
	}
	return *active
}
