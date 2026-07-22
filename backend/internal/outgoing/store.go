package outgoing

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"order-stock/backend/internal/database"
	"strings"
	"time"
)

type Store struct{ db *database.Pool }

func NewStore(db *database.Pool) *Store { return &Store{db: db} }
func tenant(a Actor) database.TenantContext {
	return database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}
}
func (s *Store) Create(ctx context.Context, a Actor, destination, notes string) (Session, error) {
	destination, e := normalizeDestination(destination)
	if e != nil {
		return Session{}, e
	}
	var id = uuid.New()
	number := "OUT-" + time.Now().Format("060102") + "-" + strings.ToUpper(uuid.NewString()[:6])
	e = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		_, e := tx.Exec(ctx, `INSERT INTO outgoing_sessions(id,tenant_id,document_number,destination,notes,created_by_user_id,updated_by_user_id) VALUES($1,$2,$3,$4,$5,$6,$6)`, id, a.TenantID, number, destination, strings.TrimSpace(notes), a.UserID)
		return e
	})
	if e != nil {
		return Session{}, e
	}
	return s.GetSession(ctx, a, id)
}
func (s *Store) GetSession(ctx context.Context, a Actor, id uuid.UUID) (Session, error) {
	var o Session
	e := database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		e := tx.QueryRow(ctx, `SELECT os.id,os.document_number,os.destination,os.notes,os.status,os.transaction_date,u.display_name FROM outgoing_sessions os JOIN users u ON u.tenant_id=os.tenant_id AND u.id=os.created_by_user_id WHERE os.tenant_id=$1 AND os.id=$2`, a.TenantID, id).Scan(&o.ID, &o.DocumentNumber, &o.Destination, &o.Notes, &o.Status, &o.TransactionDate, &o.CreatedBy)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		rows, e := tx.Query(ctx, `SELECT kl.id,kl.kanban_id,pol.raw_material_code_snapshot,pol.raw_material_name_snapshot,pol.base_unit_code_snapshot,kl.quantity FROM outgoing_session_scans oss JOIN kanban_lots kl ON kl.tenant_id=oss.tenant_id AND kl.id=oss.kanban_lot_id JOIN purchase_order_lines pol ON pol.tenant_id=kl.tenant_id AND pol.id=kl.purchase_order_line_id WHERE oss.tenant_id=$1 AND oss.session_id=$2 ORDER BY oss.scanned_at`, a.TenantID, id)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var x Scan
			x.Warehouse = "RAW MATERIAL"
			x.Location = "DEFAULT"
			if e = rows.Scan(&x.KanbanLotID, &x.KanbanID, &x.MaterialCode, &x.MaterialName, &x.Unit, &x.Quantity); e != nil {
				return e
			}
			o.Scans = append(o.Scans, x)
		}
		return rows.Err()
	})
	return o, e
}
func (s *Store) Scan(ctx context.Context, a Actor, id uuid.UUID, value string) (Session, error) {
	v, e := normalizeKanban(value)
	if e != nil {
		return Session{}, e
	}
	e = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		var status string
		if e := tx.QueryRow(ctx, `SELECT status FROM outgoing_sessions WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, a.TenantID, id).Scan(&status); e != nil {
			return ErrNotFound
		}
		if status != "ACTIVE" {
			return ErrConflict
		}
		var lot uuid.UUID
		if e := tx.QueryRow(ctx, `SELECT id FROM kanban_lots WHERE tenant_id=$1 AND kanban_id=$2 AND status='IN_STOCK' FOR UPDATE`, a.TenantID, v).Scan(&lot); errors.Is(e, pgx.ErrNoRows) {
			return ErrValidation
		} else if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO outgoing_session_scans(tenant_id,session_id,kanban_lot_id,scanned_by_user_id) VALUES($1,$2,$3,$4)`, a.TenantID, id, lot, a.UserID)
		if e != nil {
			return ErrConflict
		}
		return nil
	})
	if e != nil {
		return Session{}, e
	}
	return s.GetSession(ctx, a, id)
}
func (s *Store) Remove(ctx context.Context, a Actor, id, lot uuid.UUID) error {
	return database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		tag, e := tx.Exec(ctx, `DELETE FROM outgoing_session_scans oss USING outgoing_sessions os WHERE oss.tenant_id=$1 AND oss.session_id=$2 AND oss.kanban_lot_id=$3 AND os.tenant_id=oss.tenant_id AND os.id=oss.session_id AND os.status='ACTIVE'`, a.TenantID, id, lot)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}
func (s *Store) Complete(ctx context.Context, a Actor, id uuid.UUID) (Document, error) {
	var did uuid.UUID
	e := database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		var number, destination, notes, status string
		var date time.Time
		if e := tx.QueryRow(ctx, `SELECT document_number,destination,notes,status,transaction_date FROM outgoing_sessions WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, a.TenantID, id).Scan(&number, &destination, &notes, &status, &date); e != nil {
			return ErrNotFound
		}
		if status != "ACTIVE" {
			return ErrConflict
		}
		rows, e := tx.Query(ctx, `SELECT kl.id,pol.raw_material_id,pol.base_unit_code_snapshot FROM outgoing_session_scans oss JOIN kanban_lots kl ON kl.tenant_id=oss.tenant_id AND kl.id=oss.kanban_lot_id JOIN purchase_order_lines pol ON pol.tenant_id=kl.tenant_id AND pol.id=kl.purchase_order_line_id WHERE oss.tenant_id=$1 AND oss.session_id=$2 AND kl.status='IN_STOCK' FOR UPDATE OF kl`, a.TenantID, id)
		if e != nil {
			return e
		}
		type lot struct {
			id, material uuid.UUID
			unit         string
		}
		var lots []lot
		for rows.Next() {
			var l lot
			if e = rows.Scan(&l.id, &l.material, &l.unit); e != nil {
				rows.Close()
				return e
			}
			lots = append(lots, l)
		}
		rows.Close()
		if len(lots) == 0 {
			return ErrValidation
		}
		did = uuid.New()
		if _, e = tx.Exec(ctx, `INSERT INTO outgoing_documents(id,tenant_id,session_id,document_number,transaction_date,destination,notes,created_by_user_id,completed_by_user_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8)`, did, a.TenantID, id, number, date, destination, notes, a.UserID); e != nil {
			return e
		}
		for _, l := range lots {
			if _, e = tx.Exec(ctx, `UPDATE kanban_lots SET status='CONSUMED',updated_by_user_id=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2 AND status='IN_STOCK'`, a.TenantID, l.id, a.UserID); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO outgoing_kanban_lots(tenant_id,outgoing_id,kanban_lot_id,quantity) SELECT $1,$2,id,quantity FROM kanban_lots WHERE tenant_id=$1 AND id=$3`, a.TenantID, did, l.id); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO inventory_ledger_entries(tenant_id,event_type,kanban_lot_id,raw_material_id,quantity_delta,base_unit_code,reference_type,reference_id,created_by_user_id) SELECT $1,'OUTGOING',id,$3,-quantity,$4,'OUTGOING',$2,$5 FROM kanban_lots WHERE tenant_id=$1 AND id=$6`, a.TenantID, did, l.material, l.unit, a.UserID, l.id); e != nil {
				return e
			}
		}
		_, e = tx.Exec(ctx, `UPDATE outgoing_sessions SET status='COMPLETED',updated_by_user_id=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, a.UserID)
		return e
	})
	if e != nil {
		return Document{}, e
	}
	return s.GetDocument(ctx, a, did)
}
func (s *Store) GetDocument(ctx context.Context, a Actor, id uuid.UUID) (Document, error) {
	var o Document
	e := database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		e := tx.QueryRow(ctx, `SELECT od.id,od.document_number,od.destination,od.notes,od.status,od.transaction_date,u.display_name,COUNT(okl.kanban_lot_id),COUNT(DISTINCT pol.raw_material_id) FROM outgoing_documents od JOIN users u ON u.tenant_id=od.tenant_id AND u.id=od.completed_by_user_id JOIN outgoing_kanban_lots okl ON okl.tenant_id=od.tenant_id AND okl.outgoing_id=od.id JOIN kanban_lots kl ON kl.tenant_id=okl.tenant_id AND kl.id=okl.kanban_lot_id JOIN purchase_order_lines pol ON pol.tenant_id=kl.tenant_id AND pol.id=kl.purchase_order_line_id WHERE od.tenant_id=$1 AND od.id=$2 GROUP BY od.id,u.display_name`, a.TenantID, id).Scan(&o.ID, &o.DocumentNumber, &o.Destination, &o.Notes, &o.Status, &o.TransactionDate, &o.CreatedBy, &o.KanbanCount, &o.MaterialCount)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return e
	})
	return o, e
}
func (s *Store) List(ctx context.Context, a Actor) (items []Document, e error) {
	e = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT od.id,od.document_number,od.destination,od.notes,od.status,od.transaction_date,u.display_name,COUNT(okl.kanban_lot_id),COUNT(DISTINCT pol.raw_material_id) FROM outgoing_documents od JOIN users u ON u.tenant_id=od.tenant_id AND u.id=od.completed_by_user_id JOIN outgoing_kanban_lots okl ON okl.tenant_id=od.tenant_id AND okl.outgoing_id=od.id JOIN kanban_lots kl ON kl.tenant_id=okl.tenant_id AND kl.id=okl.kanban_lot_id JOIN purchase_order_lines pol ON pol.tenant_id=kl.tenant_id AND pol.id=kl.purchase_order_line_id WHERE od.tenant_id=$1 GROUP BY od.id,u.display_name ORDER BY od.completed_at DESC`, a.TenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var o Document
			if e = rows.Scan(&o.ID, &o.DocumentNumber, &o.Destination, &o.Notes, &o.Status, &o.TransactionDate, &o.CreatedBy, &o.KanbanCount, &o.MaterialCount); e != nil {
				return e
			}
			items = append(items, o)
		}
		return rows.Err()
	})
	return
}
