package settings

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/database"
)

type SQLStore struct{ db *database.Pool }

func NewSQLStore(db *database.Pool) *SQLStore { return &SQLStore{db: db} }

const userSelect = `SELECT u.id,u.username,u.display_name,u.email,NOT u.locked,
 EXISTS(SELECT 1 FROM tenant_settings ts WHERE ts.tenant_id=u.tenant_id AND ts.default_approver_user_id=u.id),
 COALESCE(array_agg(DISTINCT r.id) FILTER(WHERE r.id IS NOT NULL),'{}'),COALESCE(array_agg(DISTINCT r.code) FILTER(WHERE r.id IS NOT NULL),'{}'),COALESCE(array_agg(DISTINCT r.name) FILTER(WHERE r.id IS NOT NULL),'{}'),
 COALESCE((SELECT array_agg(DISTINCT rp.permission_code) FROM user_roles x JOIN roles xr ON xr.tenant_id=x.tenant_id AND xr.id=x.role_id AND xr.active JOIN role_permissions rp ON rp.tenant_id=x.tenant_id AND rp.role_id=x.role_id WHERE x.tenant_id=u.tenant_id AND x.user_id=u.id),'{}'),
 COALESCE(u.created_by_user_id,u.id),COALESCE(cu.display_name,u.display_name),u.created_at,COALESCE(u.updated_by_user_id,u.id),COALESCE(uu.display_name,u.display_name),u.updated_at
 FROM users u LEFT JOIN user_roles ur ON ur.tenant_id=u.tenant_id AND ur.user_id=u.id LEFT JOIN roles r ON r.tenant_id=ur.tenant_id AND r.id=ur.role_id
 LEFT JOIN users cu ON cu.tenant_id=u.tenant_id AND cu.id=u.created_by_user_id LEFT JOIN users uu ON uu.tenant_id=u.tenant_id AND uu.id=u.updated_by_user_id`

func scanUser(row pgx.Row) (User, error) {
	var u User
	var ids []uuid.UUID
	var codes, names []string
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Active, &u.IsPurchaseOrderApprover, &ids, &codes, &names, &u.Permissions, &u.CreatedBy, &u.CreatedByName, &u.CreatedAt, &u.UpdatedBy, &u.UpdatedByName, &u.UpdatedAt)
	for i := range ids {
		u.Roles = append(u.Roles, RoleSummary{ID: ids[i], Code: codes[i], Name: names[i]})
	}
	return u, err
}
func userGroupBy() string { return ` GROUP BY u.id,cu.display_name,uu.display_name` }
func getUser(ctx context.Context, tx database.TenantTx, tenant, id uuid.UUID) (User, error) {
	u, e := scanUser(tx.QueryRow(ctx, userSelect+` WHERE u.tenant_id=$1 AND u.id=$2`+userGroupBy(), tenant, id))
	if errors.Is(e, pgx.ErrNoRows) {
		return User{}, NotFoundError{Resource: "user"}
	}
	return u, e
}
func (s *SQLStore) ListUsers(ctx context.Context, a Actor, q ListQuery) (items []User, total int, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		var active any
		if q.Active != nil {
			active = *q.Active
		}
		filter := ` WHERE u.tenant_id=$1 AND ($2='' OR u.username ILIKE '%'||$2||'%' OR u.display_name ILIKE '%'||$2||'%' OR u.email ILIKE '%'||$2||'%') AND ($3::boolean IS NULL OR NOT u.locked=$3)`
		if e := tx.QueryRow(ctx, `SELECT count(*) FROM users u`+filter, a.TenantID, q.Search, active).Scan(&total); e != nil {
			return e
		}
		rows, e := tx.Query(ctx, userSelect+filter+userGroupBy()+` ORDER BY u.username LIMIT $4 OFFSET $5`, a.TenantID, q.Search, active, q.Limit, q.Offset)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			u, e := scanUser(rows)
			if e != nil {
				return e
			}
			items = append(items, u)
		}
		return rows.Err()
	})
	return
}
func validateRoles(ctx context.Context, tx database.TenantTx, a Actor, ids []uuid.UUID) error {
	var count int
	if e := tx.QueryRow(ctx, `SELECT count(*) FROM roles WHERE tenant_id=$1 AND active AND id=ANY($2)`, a.TenantID, ids).Scan(&count); e != nil {
		return e
	}
	if count != len(ids) {
		return ConflictError{Fields: FieldErrors{"roleIds": "Select active roles"}}
	}
	return nil
}
func assignRoles(ctx context.Context, tx database.TenantTx, a Actor, userID uuid.UUID, ids []uuid.UUID) error {
	if _, e := tx.Exec(ctx, `DELETE FROM user_roles WHERE tenant_id=$1 AND user_id=$2`, a.TenantID, userID); e != nil {
		return e
	}
	for _, id := range ids {
		if _, e := tx.Exec(ctx, `INSERT INTO user_roles(tenant_id,user_id,role_id) VALUES($1,$2,$3)`, a.TenantID, userID, id); e != nil {
			return e
		}
	}
	return nil
}
func setUserApprover(ctx context.Context, tx database.TenantTx, a Actor, userID uuid.UUID, requested bool) error {
	var current bool
	if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tenant_settings WHERE tenant_id=$1 AND default_approver_user_id=$2)`, a.TenantID, userID).Scan(&current); e != nil {
		return e
	}
	if !requested {
		if current {
			return ConflictError{Fields: FieldErrors{"isPurchaseOrderApprover": "Select another PO Approver instead"}}
		}
		return nil
	}
	var eligible bool
	if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users u JOIN user_roles ur ON ur.tenant_id=u.tenant_id AND ur.user_id=u.id JOIN roles r ON r.tenant_id=ur.tenant_id AND r.id=ur.role_id AND r.active JOIN role_permissions rp ON rp.tenant_id=r.tenant_id AND rp.role_id=r.id AND rp.permission_code='po.approve' WHERE u.tenant_id=$1 AND u.id=$2 AND NOT u.locked)`, a.TenantID, userID).Scan(&eligible); e != nil {
		return e
	}
	if !eligible {
		return ConflictError{Fields: FieldErrors{"isPurchaseOrderApprover": "Approver must be active and have PO Approve permission"}}
	}
	_, e := tx.Exec(ctx, `UPDATE tenant_settings SET default_approver_user_id=$2,updated_by_user_id=$3,updated_at=now() WHERE tenant_id=$1`, a.TenantID, userID, a.UserID)
	return e
}
func (s *SQLStore) CreateUser(ctx context.Context, a Actor, in UserInput) (item User, err error) {
	hash, e := auth.HashPassword(in.Password)
	if e != nil {
		return item, e
	}
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		if e := validateRoles(ctx, tx, a, in.RoleIDs); e != nil {
			return e
		}
		var id uuid.UUID
		e := tx.QueryRow(ctx, `INSERT INTO users(tenant_id,username,display_name,email,password_hash,locked,created_by_user_id,updated_by_user_id)VALUES($1,$2,$3,$4,$5,$6,$7,$7)RETURNING id`, a.TenantID, in.Username, in.DisplayName, in.Email, hash, !in.Active, a.UserID).Scan(&id)
		if e != nil {
			return writeError(e)
		}
		if e = assignRoles(ctx, tx, a, id, in.RoleIDs); e != nil {
			return e
		}
		if e = setUserApprover(ctx, tx, a, id, in.IsPurchaseOrderApprover); e != nil {
			return e
		}
		item, e = getUser(ctx, tx, a.TenantID, id)
		return e
	})
	return
}
func roleIDsContainAdmin(ctx context.Context, tx database.TenantTx, a Actor, ids []uuid.UUID) (bool, error) {
	var yes bool
	e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM roles WHERE tenant_id=$1 AND id=ANY($2) AND code='ADMIN')`, a.TenantID, ids).Scan(&yes)
	return yes, e
}
func (s *SQLStore) UpdateUser(ctx context.Context, a Actor, id uuid.UUID, in UserInput) (item User, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		if e := validateRoles(ctx, tx, a, in.RoleIDs); e != nil {
			return e
		}
		var wasAdmin bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.tenant_id=ur.tenant_id AND r.id=ur.role_id WHERE ur.tenant_id=$1 AND ur.user_id=$2 AND r.code='ADMIN')`, a.TenantID, id).Scan(&wasAdmin); e != nil {
			return e
		}
		willAdmin, e := roleIDsContainAdmin(ctx, tx, a, in.RoleIDs)
		if e != nil {
			return e
		}
		if wasAdmin && (!in.Active || !willAdmin) {
			var others int
			if e = tx.QueryRow(ctx, `SELECT count(DISTINCT u.id) FROM users u JOIN user_roles ur ON ur.tenant_id=u.tenant_id AND ur.user_id=u.id JOIN roles r ON r.tenant_id=ur.tenant_id AND r.id=ur.role_id WHERE u.tenant_id=$1 AND u.id<>$2 AND NOT u.locked AND r.code='ADMIN'`, a.TenantID, id).Scan(&others); e != nil {
				return e
			}
			if others == 0 {
				return ConflictError{Fields: FieldErrors{"roleIds": "At least one active Administrator is required"}}
			}
		}
		if !in.Active {
			var configured bool
			if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tenant_settings WHERE tenant_id=$1 AND default_approver_user_id=$2)`, a.TenantID, id).Scan(&configured); e != nil {
				return e
			}
			if configured {
				return ConflictError{Fields: FieldErrors{"active": "Select another Default Approver first"}}
			}
		}
		tag, e := tx.Exec(ctx, `UPDATE users SET username=$3,display_name=$4,email=$5,locked=$6,updated_by_user_id=$7,updated_at=now() WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, in.Username, in.DisplayName, in.Email, !in.Active, a.UserID)
		if e != nil {
			return writeError(e)
		}
		if tag.RowsAffected() == 0 {
			return NotFoundError{Resource: "user"}
		}
		if in.Password != "" {
			hash, e := auth.HashPassword(in.Password)
			if e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `UPDATE users SET password_hash=$3 WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, hash); e != nil {
				return e
			}
		}
		if e = assignRoles(ctx, tx, a, id, in.RoleIDs); e != nil {
			return e
		}
		if e = setUserApprover(ctx, tx, a, id, in.IsPurchaseOrderApprover); e != nil {
			return e
		}
		item, e = getUser(ctx, tx, a.TenantID, id)
		return e
	})
	return
}

const roleSelect = `SELECT r.id,r.code,r.name,r.active,r.code=ANY(ARRAY['ADMIN','DIRECTOR','PURCHASING','LOGISTICS_PLANNER','WAREHOUSE','FINANCE','VIEWER']),COALESCE(array_agg(DISTINCT rp.permission_code)FILTER(WHERE rp.permission_code IS NOT NULL),'{}'),count(DISTINCT ur.user_id),COALESCE(r.created_by_user_id,$1),COALESCE(cu.display_name,'System'),r.created_at,COALESCE(r.updated_by_user_id,$1),COALESCE(uu.display_name,'System'),r.updated_at FROM roles r LEFT JOIN role_permissions rp ON rp.tenant_id=r.tenant_id AND rp.role_id=r.id LEFT JOIN user_roles ur ON ur.tenant_id=r.tenant_id AND ur.role_id=r.id LEFT JOIN users cu ON cu.tenant_id=r.tenant_id AND cu.id=r.created_by_user_id LEFT JOIN users uu ON uu.tenant_id=r.tenant_id AND uu.id=r.updated_by_user_id`

func scanRole(row pgx.Row) (Role, error) {
	var r Role
	e := row.Scan(&r.ID, &r.Code, &r.Name, &r.Active, &r.System, &r.PermissionCodes, &r.UserCount, &r.CreatedBy, &r.CreatedByName, &r.CreatedAt, &r.UpdatedBy, &r.UpdatedByName, &r.UpdatedAt)
	return r, e
}
func roleGroup() string { return ` GROUP BY r.id,cu.display_name,uu.display_name` }
func getRole(ctx context.Context, tx database.TenantTx, a Actor, id uuid.UUID) (Role, error) {
	r, e := scanRole(tx.QueryRow(ctx, roleSelect+` WHERE r.tenant_id=$2 AND r.id=$3`+roleGroup(), a.UserID, a.TenantID, id))
	if errors.Is(e, pgx.ErrNoRows) {
		return Role{}, NotFoundError{Resource: "role"}
	}
	return r, e
}
func (s *SQLStore) ListRoles(ctx context.Context, a Actor, q ListQuery) (items []Role, total int, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		var active any
		if q.Active != nil {
			active = *q.Active
		}
		filter := ` WHERE r.tenant_id=$2 AND ($3='' OR r.code ILIKE '%'||$3||'%' OR r.name ILIKE '%'||$3||'%') AND ($4::boolean IS NULL OR r.active=$4)`
		if e := tx.QueryRow(ctx, `SELECT count(*) FROM roles r WHERE r.tenant_id=$1 AND ($2='' OR r.code ILIKE '%'||$2||'%' OR r.name ILIKE '%'||$2||'%') AND ($3::boolean IS NULL OR r.active=$3)`, a.TenantID, q.Search, active).Scan(&total); e != nil {
			return e
		}
		rows, e := tx.Query(ctx, roleSelect+filter+roleGroup()+` ORDER BY r.code LIMIT $5 OFFSET $6`, a.UserID, a.TenantID, q.Search, active, q.Limit, q.Offset)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			r, e := scanRole(rows)
			if e != nil {
				return e
			}
			items = append(items, r)
		}
		return rows.Err()
	})
	return
}
func replacePermissions(ctx context.Context, tx database.TenantTx, a Actor, id uuid.UUID, codes []string) error {
	if _, e := tx.Exec(ctx, `DELETE FROM role_permissions WHERE tenant_id=$1 AND role_id=$2`, a.TenantID, id); e != nil {
		return e
	}
	for _, code := range codes {
		if _, e := tx.Exec(ctx, `INSERT INTO role_permissions(tenant_id,role_id,permission_code)VALUES($1,$2,$3)`, a.TenantID, id, code); e != nil {
			return ConflictError{Fields: FieldErrors{"permissionCodes": "Select valid permissions"}}
		}
	}
	return nil
}
func (s *SQLStore) CreateRole(ctx context.Context, a Actor, in RoleInput) (item Role, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		var id uuid.UUID
		e := tx.QueryRow(ctx, `INSERT INTO roles(tenant_id,code,name,active,created_by_user_id,updated_by_user_id)VALUES($1,$2,$3,$4,$5,$5)RETURNING id`, a.TenantID, in.Code, in.Name, in.Active, a.UserID).Scan(&id)
		if e != nil {
			return writeError(e)
		}
		if e = replacePermissions(ctx, tx, a, id, in.PermissionCodes); e != nil {
			return e
		}
		item, e = getRole(ctx, tx, a, id)
		return e
	})
	return
}
func (s *SQLStore) UpdateRole(ctx context.Context, a Actor, id uuid.UUID, in RoleInput) (item Role, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		current, e := getRole(ctx, tx, a, id)
		if e != nil {
			return e
		}
		keepsApprove := false
		for _, p := range in.PermissionCodes {
			if p == "po.approve" {
				keepsApprove = true
			}
		}
		if !in.Active || !keepsApprove {
			var blocks bool
			e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tenant_settings ts JOIN user_roles ur ON ur.tenant_id=ts.tenant_id AND ur.user_id=ts.default_approver_user_id WHERE ts.tenant_id=$1 AND ur.role_id=$2 AND NOT EXISTS(SELECT 1 FROM user_roles ur2 JOIN roles r2 ON r2.tenant_id=ur2.tenant_id AND r2.id=ur2.role_id AND r2.active JOIN role_permissions rp2 ON rp2.tenant_id=r2.tenant_id AND rp2.role_id=r2.id AND rp2.permission_code='po.approve' WHERE ur2.tenant_id=ts.tenant_id AND ur2.user_id=ts.default_approver_user_id AND ur2.role_id<>$2))`, a.TenantID, id).Scan(&blocks)
			if e != nil {
				return e
			}
			if blocks {
				return ConflictError{Fields: FieldErrors{"permissionCodes": "Select another Default Approver first"}}
			}
		}
		code := in.Code
		if current.System {
			code = current.Code
		}
		_, e = tx.Exec(ctx, `UPDATE roles SET code=$3,name=$4,active=$5,updated_by_user_id=$6,updated_at=now() WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, code, in.Name, in.Active, a.UserID)
		if e != nil {
			return writeError(e)
		}
		if e = replacePermissions(ctx, tx, a, id, in.PermissionCodes); e != nil {
			return e
		}
		item, e = getRole(ctx, tx, a, id)
		return e
	})
	return
}
func (s *SQLStore) ListPermissions(ctx context.Context, a Actor) (items []Permission, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT code,description FROM permissions ORDER BY code`)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var p Permission
			if e = rows.Scan(&p.Code, &p.Description); e != nil {
				return e
			}
			p.Group = permissionGroup(p.Code)
			items = append(items, p)
		}
		return rows.Err()
	})
	return
}
func (s *SQLStore) GetApprovalConfig(ctx context.Context, a Actor) (item ApprovalConfig, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		e := tx.QueryRow(ctx, `SELECT COALESCE(ts.default_approver_user_id,'00000000-0000-0000-0000-000000000000'),COALESCE(u.display_name,''),COALESCE(u.email,'') FROM tenant_settings ts LEFT JOIN users u ON u.tenant_id=ts.tenant_id AND u.id=ts.default_approver_user_id WHERE ts.tenant_id=$1`, a.TenantID).Scan(&item.DefaultApproverUserID, &item.DisplayName, &item.Email)
		return e
	})
	return
}
func (s *SQLStore) UpdateApprovalConfig(ctx context.Context, a Actor, in ApprovalConfigInput) (item ApprovalConfig, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		var eligible bool
		e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users u JOIN user_roles ur ON ur.tenant_id=u.tenant_id AND ur.user_id=u.id JOIN roles r ON r.tenant_id=ur.tenant_id AND r.id=ur.role_id AND r.active JOIN role_permissions rp ON rp.tenant_id=r.tenant_id AND rp.role_id=r.id AND rp.permission_code='po.approve' WHERE u.tenant_id=$1 AND u.id=$2 AND NOT u.locked)`, a.TenantID, in.DefaultApproverUserID).Scan(&eligible)
		if e != nil {
			return e
		}
		if !eligible {
			return ConflictError{Fields: FieldErrors{"defaultApproverUserId": "Select an active user with PO Approve permission"}}
		}
		_, e = tx.Exec(ctx, `UPDATE tenant_settings SET default_approver_user_id=$2,updated_by_user_id=$3,updated_at=now() WHERE tenant_id=$1`, a.TenantID, in.DefaultApproverUserID, a.UserID)
		if e != nil {
			return e
		}
		e = tx.QueryRow(ctx, `SELECT u.id,u.display_name,u.email FROM users u WHERE u.tenant_id=$1 AND u.id=$2`, a.TenantID, in.DefaultApproverUserID).Scan(&item.DefaultApproverUserID, &item.DisplayName, &item.Email)
		return e
	})
	return
}
func (s *SQLStore) GetCompanyConfig(ctx context.Context, a Actor) (item CompanyConfig, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		var logo, background bool
		if err := tx.QueryRow(ctx, `SELECT company_name,company_logo IS NOT NULL,login_background IS NOT NULL FROM tenant_settings WHERE tenant_id=$1`, a.TenantID).Scan(&item.CompanyName, &logo, &background); err != nil {
			return err
		}
		if logo {
			item.LogoURL = mediaURL("logo")
		}
		if background {
			item.LoginBackgroundURL = mediaURL("login-background")
		}
		return nil
	})
	return
}
func (s *SQLStore) UpdateCompanyMedia(ctx context.Context, a Actor, kind string, in BrandingInput) (item CompanyConfig, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		if kind == "logo" {
			_, err = tx.Exec(ctx, `UPDATE tenant_settings SET company_logo=$2,company_logo_mime=$3,updated_by_user_id=$4,updated_at=now() WHERE tenant_id=$1`, a.TenantID, in.Content, in.ContentType, a.UserID)
		} else {
			_, err = tx.Exec(ctx, `UPDATE tenant_settings SET login_background=$2,login_background_mime=$3,updated_by_user_id=$4,updated_at=now() WHERE tenant_id=$1`, a.TenantID, in.Content, in.ContentType, a.UserID)
		}
		return err
	})
	if err == nil {
		item, err = s.GetCompanyConfig(ctx, a)
	}
	return
}
func (s *SQLStore) DeleteCompanyMedia(ctx context.Context, a Actor, kind string) (item CompanyConfig, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		if kind == "logo" {
			_, err = tx.Exec(ctx, `UPDATE tenant_settings SET company_logo=NULL,company_logo_mime=NULL,updated_by_user_id=$2,updated_at=now() WHERE tenant_id=$1`, a.TenantID, a.UserID)
		} else {
			_, err = tx.Exec(ctx, `UPDATE tenant_settings SET login_background=NULL,login_background_mime=NULL,updated_by_user_id=$2,updated_at=now() WHERE tenant_id=$1`, a.TenantID, a.UserID)
		}
		return err
	})
	if err == nil {
		item, err = s.GetCompanyConfig(ctx, a)
	}
	return
}
func (s *SQLStore) GetPublicBranding(ctx context.Context) (item CompanyConfig, err error) {
	var logo, background bool
	err = s.db.QueryRow(ctx, `SELECT company_name,has_logo,has_login_background FROM public_branding()`).Scan(&item.CompanyName, &logo, &background)
	if logo {
		item.LogoURL = mediaURL("logo")
	}
	if background {
		item.LoginBackgroundURL = mediaURL("login-background")
	}
	return
}
func (s *SQLStore) GetPublicBrandingMedia(ctx context.Context, kind string) (item BrandingMedia, err error) {
	err = s.db.QueryRow(ctx, `SELECT content,content_type FROM public_branding_media($1)`, kind).Scan(&item.Content, &item.ContentType)
	if err == nil && len(item.Content) == 0 {
		err = NotFoundError{"branding media"}
	}
	return
}
func (s *SQLStore) UpdateCompanyConfig(ctx context.Context, a Actor, in CompanyConfigInput) (item CompanyConfig, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}, func(tx database.TenantTx) error {
		_, err := tx.Exec(ctx, `UPDATE tenant_settings SET company_name=$2,updated_by_user_id=$3,updated_at=now() WHERE tenant_id=$1`, a.TenantID, in.CompanyName, a.UserID)
		return err
	})
	if err == nil {
		item, err = s.GetCompanyConfig(ctx, a)
	}
	return
}
func writeError(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) && p.Code == "23505" {
		field := "username"
		if strings.Contains(p.ConstraintName, "email") {
			field = "email"
		}
		if strings.Contains(p.ConstraintName, "roles") {
			field = "code"
		}
		return ConflictError{Fields: FieldErrors{field: "Already in use"}}
	}
	return err
}
