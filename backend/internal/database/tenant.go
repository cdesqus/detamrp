package database

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type TenantContext struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

type TenantTx interface {
	SetTenant(ctx context.Context, tenantID string) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type TenantBeginner interface {
	BeginTenantTx(ctx context.Context) (TenantTx, error)
}

func WithTenant(ctx context.Context, beginner TenantBeginner, tenant TenantContext, callback func(TenantTx) error) error {
	if tenant.TenantID == uuid.Nil {
		return errors.New("tenant ID is required")
	}
	tx, err := beginner.BeginTenantTx(ctx)
	if err != nil {
		return err
	}
	if err := tx.SetTenant(ctx, tenant.TenantID.String()); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := callback(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}
