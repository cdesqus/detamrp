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

const measurementSelect = `SELECT m.id, m.code, m.name, m.decimal_allowed, m.active,
 m.created_by_user_id, cu.display_name, m.created_at, m.updated_by_user_id, uu.display_name, m.updated_at
 FROM measurements m JOIN users cu ON cu.tenant_id=m.tenant_id AND cu.id=m.created_by_user_id
 JOIN users uu ON uu.tenant_id=m.tenant_id AND uu.id=m.updated_by_user_id`

func scanMeasurement(row pgx.Row) (Measurement, error) {
	var m Measurement
	err := row.Scan(&m.ID, &m.Code, &m.Name, &m.DecimalAllowed, &m.Active, &m.CreatedBy,
		&m.CreatedByName, &m.CreatedAt, &m.UpdatedBy, &m.UpdatedByName, &m.UpdatedAt)
	return m, err
}

func (s *SQLStore) ListMeasurements(ctx context.Context, actor Actor, q ListQuery) (items []Measurement, total int, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		var active any
		if q.Active != nil {
			active = *q.Active
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM measurements m WHERE m.tenant_id=$1 AND
 ($2='' OR m.code ILIKE '%'||$2||'%' OR m.name ILIKE '%'||$2||'%') AND ($3::boolean IS NULL OR m.active=$3)`, actor.TenantID, q.Search, active).Scan(&total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, measurementSelect+` WHERE m.tenant_id=$1 AND
 ($2='' OR m.code ILIKE '%'||$2||'%' OR m.name ILIKE '%'||$2||'%') AND ($3::boolean IS NULL OR m.active=$3)
 ORDER BY m.code LIMIT $4 OFFSET $5`, actor.TenantID, q.Search, active, q.Limit, q.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			m, err := scanMeasurement(rows)
			if err != nil {
				return err
			}
			items = append(items, m)
		}
		return rows.Err()
	})
	return
}

func (s *SQLStore) GetMeasurement(ctx context.Context, actor Actor, id uuid.UUID) (item Measurement, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		var scanErr error
		item, scanErr = scanMeasurement(tx.QueryRow(ctx, measurementSelect+` WHERE m.tenant_id=$1 AND m.id=$2`, actor.TenantID, id))
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return NotFoundError{Resource: "measurement"}
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

func (s *SQLStore) CreateMeasurement(ctx context.Context, actor Actor, in MeasurementInput) (item Measurement, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `INSERT INTO measurements (tenant_id,code,name,decimal_allowed,active,created_by_user_id,updated_by_user_id)
 VALUES ($1,$2,$3,$4,$5,$6,$6) RETURNING id`, actor.TenantID, in.Code, in.Name, in.DecimalAllowed, activeValue(in.Active), actor.UserID).Scan(&id)
		if err != nil {
			return masterDataWriteError(err, "code")
		}
		item, err = scanMeasurement(tx.QueryRow(ctx, measurementSelect+` WHERE m.tenant_id=$1 AND m.id=$2`, actor.TenantID, id))
		return err
	})
	return
}

func (s *SQLStore) UpdateMeasurement(ctx context.Context, actor Actor, id uuid.UUID, in MeasurementInput) (item Measurement, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		tag, err := tx.Exec(ctx, `UPDATE measurements SET code=$3,name=$4,decimal_allowed=$5,active=$6,updated_by_user_id=$7,updated_at=now()
 WHERE tenant_id=$1 AND id=$2`, actor.TenantID, id, in.Code, in.Name, in.DecimalAllowed, activeValue(in.Active), actor.UserID)
		if err != nil {
			return masterDataWriteError(err, "code")
		}
		if tag.RowsAffected() == 0 {
			return NotFoundError{Resource: "measurement"}
		}
		item, err = scanMeasurement(tx.QueryRow(ctx, measurementSelect+` WHERE m.tenant_id=$1 AND m.id=$2`, actor.TenantID, id))
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
