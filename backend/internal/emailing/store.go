package emailing

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/database"
)

type Store struct{ db *database.Pool }

func NewStore(db *database.Pool) *Store { return &Store{db: db} }

type storedSMTP struct {
	SMTPSettings
	PasswordEncrypted []byte
}

func (s *Store) GetSMTP(ctx context.Context, tenantID uuid.UUID) (out storedSMTP, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: tenantID}, func(tx database.TenantTx) error {
		return tx.QueryRow(ctx, `SELECT smtp_host,smtp_port,smtp_security,smtp_username,COALESCE(smtp_password_encrypted,''::bytea),smtp_from_name,smtp_from_email FROM tenant_settings WHERE tenant_id=$1`, tenantID).
			Scan(&out.Host, &out.Port, &out.Security, &out.Username, &out.PasswordEncrypted, &out.FromName, &out.FromEmail)
	})
	out.PasswordSet = len(out.PasswordEncrypted) > 0
	return
}

func (s *Store) SaveSMTP(ctx context.Context, actor Actor, in SMTPSettingsInput, encrypted []byte, keepPassword bool) error {
	return database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		if keepPassword {
			_, err := tx.Exec(ctx, `UPDATE tenant_settings SET smtp_host=$2,smtp_port=$3,smtp_security=$4,smtp_username=$5,smtp_from_name=$6,smtp_from_email=$7,updated_by_user_id=$8,updated_at=now() WHERE tenant_id=$1`,
				actor.TenantID, in.Host, in.Port, in.Security, in.Username, in.FromName, in.FromEmail, actor.UserID)
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE tenant_settings SET smtp_host=$2,smtp_port=$3,smtp_security=$4,smtp_username=$5,smtp_password_encrypted=$6,smtp_from_name=$7,smtp_from_email=$8,updated_by_user_id=$9,updated_at=now() WHERE tenant_id=$1`,
			actor.TenantID, in.Host, in.Port, in.Security, in.Username, encrypted, in.FromName, in.FromEmail, actor.UserID)
		return err
	})
}

func (s *Store) CreateLog(ctx context.Context, actor Actor, typ, refType string, refID uuid.UUID, refNumber, recipient, subject string) (id uuid.UUID, err error) {
	var nullableRefID any
	if refID != uuid.Nil {
		nullableRefID = refID
	}
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		return tx.QueryRow(ctx, `INSERT INTO email_logs(tenant_id,email_type,reference_type,reference_id,reference_number,recipient,subject,status,created_by_user_id) VALUES($1,$2,$3,$4,$5,$6,$7,'PENDING',$8) RETURNING id`,
			actor.TenantID, typ, refType, nullableRefID, refNumber, recipient, subject, actor.UserID).Scan(&id)
	})
	return
}
func (s *Store) FinishLog(ctx context.Context, tenantID, id uuid.UUID, sendErr error) error {
	return database.WithTenant(ctx, s.db, database.TenantContext{TenantID: tenantID}, func(tx database.TenantTx) error {
		if sendErr == nil {
			_, e := tx.Exec(ctx, `UPDATE email_logs SET status='SENT',attempts=attempts+1,last_error='',sent_at=now(),updated_at=now() WHERE tenant_id=$1 AND id=$2`, tenantID, id)
			return e
		}
		message := sendErr.Error()
		if len(message) > 500 {
			message = message[:500]
		}
		_, e := tx.Exec(ctx, `UPDATE email_logs SET status='FAILED',attempts=attempts+1,last_error=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2`, tenantID, id, message)
		return e
	})
}
func (s *Store) ListLogs(ctx context.Context, actor Actor, search string) (items []EmailLog, err error) {
	search = strings.TrimSpace(search)
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID}, func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT id,email_type,reference_number,recipient,subject,status,attempts,last_error,sent_at,created_at FROM email_logs WHERE tenant_id=$1 AND ($2='' OR recipient ILIKE '%'||$2||'%' OR reference_number ILIKE '%'||$2||'%' OR subject ILIKE '%'||$2||'%') ORDER BY created_at DESC LIMIT 200`, actor.TenantID, search)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var x EmailLog
			if e = rows.Scan(&x.ID, &x.EmailType, &x.ReferenceNumber, &x.Recipient, &x.Subject, &x.Status, &x.Attempts, &x.LastError, &x.SentAt, &x.CreatedAt); e != nil {
				return e
			}
			items = append(items, x)
		}
		return rows.Err()
	})
	return
}

func (s *Store) ApprovalData(ctx context.Context, actor Actor, poID uuid.UUID) (out ApprovalMailData, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID}, func(tx database.TenantTx) error {
		err := tx.QueryRow(ctx, `SELECT a.id,p.id,a.approver_user_id,p.po_number,s.name,a.approver_email,a.approver_display_name,cu.display_name,p.order_date,p.expected_delivery_date,p.currency,p.total_amount,p.notes
		 FROM purchase_orders p JOIN suppliers s ON s.tenant_id=p.tenant_id AND s.id=p.supplier_id JOIN purchase_order_approvals a ON a.tenant_id=p.tenant_id AND a.purchase_order_id=p.id AND a.status='PENDING'
		 JOIN users cu ON cu.tenant_id=p.tenant_id AND cu.id=p.created_by_user_id WHERE p.tenant_id=$1 AND p.id=$2 ORDER BY a.version DESC LIMIT 1`,
			actor.TenantID, poID).Scan(&out.ApprovalID, &out.PurchaseOrderID, &out.ApproverUserID, &out.PONumber, &out.SupplierName, &out.ApproverEmail, &out.ApproverName, &out.CreatedByName, &out.OrderDate, &out.ExpectedDeliveryDate, &out.Currency, &out.TotalAmount, &out.Notes)
		if err != nil {
			return err
		}
		rows, e := tx.Query(ctx, `SELECT l.raw_material_code_snapshot,l.raw_material_name_snapshot,l.base_unit_code_snapshot,l.qty_per_kanban_snapshot,l.total_kanban,l.ordered_base_qty FROM purchase_order_lines l WHERE l.tenant_id=$1 AND l.purchase_order_id=$2 ORDER BY l.sort_position`, actor.TenantID, poID)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var x ApprovalMailLine
			var kanban decimal.Decimal
			if e = rows.Scan(&x.Code, &x.Name, &x.Unit, &x.QtyPerKanban, &kanban, &x.TotalQuantity); e != nil {
				return e
			}
			x.TotalKanban = kanban.IntPart()
			out.Lines = append(out.Lines, x)
		}
		return rows.Err()
	})
	return
}

func (s *Store) SaveApprovalToken(ctx context.Context, tenantID, approvalID uuid.UUID, hash []byte, expires time.Time) error {
	return database.WithTenant(ctx, s.db, database.TenantContext{TenantID: tenantID}, func(tx database.TenantTx) error {
		_, e := tx.Exec(ctx, `UPDATE approval_email_tokens SET used_at=now() WHERE tenant_id=$1 AND approval_id=$2 AND used_at IS NULL`, tenantID, approvalID)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO approval_email_tokens(tenant_id,approval_id,token_hash,expires_at) VALUES($1,$2,$3,$4)`, tenantID, approvalID, hash, expires)
		return e
	})
}

type TokenContext struct {
	TenantID, ApprovalID, ApproverUserID uuid.UUID
	ApproverName                         string
}

func (s *Store) ResolveToken(ctx context.Context, tenantID uuid.UUID, hash []byte) (out TokenContext, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: tenantID}, func(tx database.TenantTx) error {
		return tx.QueryRow(ctx, `SELECT t.tenant_id,t.approval_id,a.approver_user_id,a.approver_display_name FROM approval_email_tokens t JOIN purchase_order_approvals a ON a.tenant_id=t.tenant_id AND a.id=t.approval_id WHERE t.tenant_id=$1 AND t.token_hash=$2 AND t.used_at IS NULL AND t.expires_at>now() AND a.status='PENDING'`, tenantID, hash).Scan(&out.TenantID, &out.ApprovalID, &out.ApproverUserID, &out.ApproverName)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		err = errors.New("approval link is invalid, expired, or already used")
	}
	return
}
func (s *Store) MarkTokenUsed(ctx context.Context, tenantID uuid.UUID, hash []byte) error {
	return database.WithTenant(ctx, s.db, database.TenantContext{TenantID: tenantID}, func(tx database.TenantTx) error {
		_, e := tx.Exec(ctx, `UPDATE approval_email_tokens SET used_at=now() WHERE tenant_id=$1 AND token_hash=$2 AND used_at IS NULL`, tenantID, hash)
		return e
	})
}

func (s *Store) SupplierEmail(ctx context.Context, tenantID, supplierID uuid.UUID) (email string, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: tenantID}, func(tx database.TenantTx) error {
		return tx.QueryRow(ctx, `SELECT email FROM suppliers WHERE tenant_id=$1 AND id=$2 AND active`, tenantID, supplierID).Scan(&email)
	})
	return
}
