package auth

import (
	"context"

	"github.com/google/uuid"
	"order-stock/backend/internal/database"
)

type SQLStore struct {
	database *database.Pool
	tenantID uuid.UUID
}

func NewSQLStore(db *database.Pool, tenantID uuid.UUID) *SQLStore {
	return &SQLStore{database: db, tenantID: tenantID}
}

func (s *SQLStore) FindUserByUsername(ctx context.Context, username string) (User, error) {
	var user User
	err := database.WithTenant(ctx, s.database, database.TenantContext{TenantID: s.tenantID}, func(tx database.TenantTx) error {
		return tx.QueryRow(ctx, `
			SELECT u.id, u.tenant_id, u.username, u.display_name, u.password_hash, u.locked,
			       ARRAY(SELECT DISTINCT rp.permission_code FROM user_roles ur JOIN role_permissions rp ON rp.tenant_id = ur.tenant_id AND rp.role_id = ur.role_id WHERE ur.tenant_id = u.tenant_id AND ur.user_id = u.id)
			FROM users
			WHERE u.tenant_id = $1 AND lower(u.username) = lower($2)
		`, s.tenantID, username).Scan(&user.ID, &user.TenantID, &user.Username, &user.DisplayName, &user.PasswordHash, &user.Locked, &user.Permissions)
	})
	return user, err
}

func (s *SQLStore) CreateSession(ctx context.Context, session Session) error {
	return database.WithTenant(ctx, s.database, database.TenantContext{TenantID: s.tenantID, UserID: session.UserID}, func(tx database.TenantTx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO sessions (id, tenant_id, user_id, token_hash, expires_at)
			VALUES ($1, $2, $3, $4, $5)
		`, session.ID, session.TenantID, session.UserID, session.TokenHash, session.ExpiresAt)
		return err
	})
}

func (s *SQLStore) FindSessionByTokenHash(ctx context.Context, tokenHash string) (SessionUser, error) {
	var result SessionUser
	err := database.WithTenant(ctx, s.database, database.TenantContext{TenantID: s.tenantID}, func(tx database.TenantTx) error {
		return tx.QueryRow(ctx, `
			SELECT s.id, s.tenant_id, s.user_id, s.token_hash, s.expires_at,
			       u.id, u.tenant_id, u.username, u.display_name, u.password_hash, u.locked,
			       ARRAY(SELECT DISTINCT rp.permission_code FROM user_roles ur JOIN role_permissions rp ON rp.tenant_id = ur.tenant_id AND rp.role_id = ur.role_id WHERE ur.tenant_id = u.tenant_id AND ur.user_id = u.id)
			FROM sessions s
			JOIN users u ON u.tenant_id = s.tenant_id AND u.id = s.user_id
			WHERE s.tenant_id = $1 AND s.token_hash = $2
		`, s.tenantID, tokenHash).Scan(
			&result.Session.ID, &result.Session.TenantID, &result.Session.UserID, &result.Session.TokenHash, &result.Session.ExpiresAt,
			&result.User.ID, &result.User.TenantID, &result.User.Username, &result.User.DisplayName, &result.User.PasswordHash, &result.User.Locked, &result.User.Permissions,
		)
	})
	return result, err
}

func (s *SQLStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	return database.WithTenant(ctx, s.database, database.TenantContext{TenantID: s.tenantID}, func(tx database.TenantTx) error {
		_, err := tx.Exec(ctx, `DELETE FROM sessions WHERE tenant_id = $1 AND token_hash = $2`, s.tenantID, tokenHash)
		return err
	})
}

func (s *SQLStore) EnsureInitialAdmin(ctx context.Context, username, password string) error {
	passwordHash, err := HashPassword(password)
	if err != nil {
		return err
	}
	return database.WithTenant(ctx, s.database, database.TenantContext{TenantID: s.tenantID}, func(tx database.TenantTx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO users (tenant_id, username, display_name, password_hash)
			VALUES ($1, $2, 'Administrator', $3)
			ON CONFLICT (tenant_id, username) DO NOTHING
		`, s.tenantID, username, passwordHash)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO user_roles (tenant_id, user_id, role_id)
			SELECT $1, u.id, r.id FROM users u JOIN roles r ON r.tenant_id = u.tenant_id AND r.code = 'ADMIN'
			WHERE u.tenant_id = $1 AND u.username = $2
			ON CONFLICT DO NOTHING
		`, s.tenantID, username)
		return err
	})
}
