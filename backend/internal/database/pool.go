package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct{ pool *pgxpool.Pool }

func Open(ctx context.Context, databaseURL string) (*Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Pool{pool: pool}, nil
}

func (p *Pool) Close() { p.pool.Close() }

func (p *Pool) BeginTenantTx(ctx context.Context) (TenantTx, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxTenantTx{tx: tx}, nil
}

type pgxTenantTx struct{ tx pgx.Tx }

func (p *pgxTenantTx) SetTenant(ctx context.Context, tenantID string) error {
	_, err := p.tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, tenantID)
	return err
}
func (p *pgxTenantTx) Commit(ctx context.Context) error   { return p.tx.Commit(ctx) }
func (p *pgxTenantTx) Rollback(ctx context.Context) error { return p.tx.Rollback(ctx) }
