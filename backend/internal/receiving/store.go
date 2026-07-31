package receiving

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"order-stock/backend/internal/database"
)

type Store struct{ db *database.Pool }

func NewStore(db *database.Pool) *Store { return &Store{db: db} }
func tenant(a Actor) database.TenantContext {
	return database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}
}

func (s *Store) Options(ctx context.Context, a Actor, search string) (result []Option, err error) {
	err = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT dn.id,dn.delivery_note_number,p.po_number,s.name,COUNT(kl.id),COUNT(*) FILTER(WHERE kl.status IN ('IN_STOCK','CONSUMED')),COUNT(*) FILTER(WHERE kl.status='ISSUED') FROM delivery_notes dn JOIN purchase_orders p ON p.tenant_id=dn.tenant_id AND p.id=dn.purchase_order_id JOIN suppliers s ON s.tenant_id=p.tenant_id AND s.id=p.supplier_id JOIN delivery_note_lines dnl ON dnl.tenant_id=dn.tenant_id AND dnl.delivery_note_id=dn.id JOIN kanban_lots kl ON kl.tenant_id=dnl.tenant_id AND kl.delivery_note_line_id=dnl.id WHERE dn.tenant_id=$1 AND ($2='' OR dn.delivery_note_number ILIKE '%'||$2||'%' OR p.po_number ILIKE '%'||$2||'%') GROUP BY dn.id,dn.delivery_note_number,p.po_number,s.name HAVING COUNT(*) FILTER(WHERE kl.status='ISSUED')>0 ORDER BY dn.delivery_note_number`, a.TenantID, search)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var o Option
			if e = rows.Scan(&o.DeliveryNoteID, &o.DeliveryNoteNumber, &o.PONumber, &o.SupplierName, &o.Planned, &o.Received, &o.Outstanding); e != nil {
				return e
			}
			result = append(result, o)
		}
		return rows.Err()
	})
	return
}

func (s *Store) CreateSession(ctx context.Context, a Actor, deliveryNoteNumber string) (Session, error) {
	deliveryNoteNumber, err := normalizeDeliveryNoteNumber(deliveryNoteNumber)
	if err != nil {
		return Session{}, ErrDeliveryNoteInvalid
	}
	var out Session
	err = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		_, _ = tx.Exec(ctx, `UPDATE receiving_sessions SET status='EXPIRED',updated_at=now() WHERE tenant_id=$1 AND status='ACTIVE' AND expires_at<now()`, a.TenantID)
		var dnID uuid.UUID
		var poStatus string
		lookupErr := tx.QueryRow(ctx, `SELECT dn.id,p.status FROM delivery_notes dn JOIN purchase_orders p ON p.tenant_id=dn.tenant_id AND p.id=dn.purchase_order_id WHERE dn.tenant_id=$1 AND upper(dn.delivery_note_number)=$2 FOR UPDATE OF dn`, a.TenantID, deliveryNoteNumber).Scan(&dnID, &poStatus)
		if errors.Is(lookupErr, pgx.ErrNoRows) || (lookupErr == nil && poStatus != "APPROVED" && poStatus != "PARTIALLY_RECEIVED") {
			return ErrDeliveryNoteInvalid
		}
		if lookupErr != nil {
			return lookupErr
		}
		var outstanding int
		if e := tx.QueryRow(ctx, `SELECT COUNT(*) FILTER(WHERE kl.status='ISSUED') FROM delivery_note_lines dnl JOIN kanban_lots kl ON kl.tenant_id=dnl.tenant_id AND kl.delivery_note_line_id=dnl.id WHERE dnl.tenant_id=$1 AND dnl.delivery_note_id=$2`, a.TenantID, dnID).Scan(&outstanding); e != nil {
			return e
		}
		if outstanding == 0 {
			return ErrDeliveryNoteFullyReceived
		}
		var open bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM receiving_sessions WHERE tenant_id=$1 AND delivery_note_id=$2 AND status IN ('ACTIVE','PAUSED'))`, a.TenantID, dnID).Scan(&open); e != nil {
			return e
		}
		if open {
			return ErrDeliveryNoteInProgress
		}
		number := "RCV-" + time.Now().Format("060102") + "-" + stringsUpper(uuid.NewString()[:6])
		id := uuid.New()
		insertErr := tx.QueryRow(ctx, `INSERT INTO receiving_sessions(id,tenant_id,delivery_note_id,receiving_number,created_by_user_id,updated_by_user_id) VALUES($1,$2,$3,$4,$5,$5) RETURNING id,delivery_note_id,receiving_number,status,receiving_date`, id, a.TenantID, dnID, number, a.UserID).Scan(&out.ID, &out.DeliveryNoteID, &out.ReceivingNumber, &out.Status, &out.ReceivingDate)
		if insertErr != nil {
			return ErrDeliveryNoteInProgress
		}
		return nil
	})
	if err != nil {
		return Session{}, err
	}
	return s.GetSession(ctx, a, out.ID)
}
func stringsUpper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}

func (s *Store) GetSession(ctx context.Context, a Actor, id uuid.UUID) (Session, error) {
	var out Session
	err := database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error { return loadSession(ctx, tx, a, id, &out) })
	return out, err
}

func (s *Store) ListOpenSessions(ctx context.Context, a Actor) (items []Session, err error) {
	var ids []uuid.UUID
	err = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT id FROM receiving_sessions WHERE tenant_id=$1 AND status IN ('ACTIVE','PAUSED') ORDER BY updated_at DESC`, a.TenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var id uuid.UUID
			if e = rows.Scan(&id); e != nil {
				return e
			}
			ids = append(ids, id)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		item, e := s.GetSession(ctx, a, id)
		if e != nil {
			return nil, e
		}
		items = append(items, item)
	}
	return items, nil
}
func loadSession(ctx context.Context, tx database.TenantTx, a Actor, id uuid.UUID, out *Session) error {
	out.Scans = []Scan{}
	err := tx.QueryRow(ctx, `SELECT rs.id,rs.delivery_note_id,rs.receiving_number,rs.status,rs.receiving_date,dn.delivery_note_number,p.po_number,s.name,u.display_name,COUNT(kl.id),COUNT(*) FILTER(WHERE kl.status IN ('IN_STOCK','CONSUMED')),COUNT(*) FILTER(WHERE kl.status='ISSUED') FROM receiving_sessions rs JOIN delivery_notes dn ON dn.tenant_id=rs.tenant_id AND dn.id=rs.delivery_note_id JOIN purchase_orders p ON p.tenant_id=dn.tenant_id AND p.id=dn.purchase_order_id JOIN suppliers s ON s.tenant_id=p.tenant_id AND s.id=p.supplier_id JOIN users u ON u.tenant_id=rs.tenant_id AND u.id=rs.created_by_user_id JOIN delivery_note_lines dnl ON dnl.tenant_id=dn.tenant_id AND dnl.delivery_note_id=dn.id JOIN kanban_lots kl ON kl.tenant_id=dnl.tenant_id AND kl.delivery_note_line_id=dnl.id WHERE rs.tenant_id=$1 AND rs.id=$2 GROUP BY rs.id,dn.delivery_note_number,p.po_number,s.name,u.display_name`, a.TenantID, id).Scan(&out.ID, &out.DeliveryNoteID, &out.ReceivingNumber, &out.Status, &out.ReceivingDate, &out.DeliveryNoteNumber, &out.PONumber, &out.SupplierName, &out.CreatedBy, &out.Planned, &out.PreviouslyReceived, &out.Outstanding)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	rows, e := tx.Query(ctx, `SELECT kl.id,kl.kanban_id,pol.raw_material_code_snapshot,pol.raw_material_name_snapshot,pol.base_unit_code_snapshot,kl.quantity FROM receiving_session_scans rss JOIN kanban_lots kl ON kl.tenant_id=rss.tenant_id AND kl.id=rss.kanban_lot_id JOIN purchase_order_lines pol ON pol.tenant_id=kl.tenant_id AND pol.id=kl.purchase_order_line_id WHERE rss.tenant_id=$1 AND rss.session_id=$2 ORDER BY rss.scanned_at`, a.TenantID, id)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var x Scan
		if e = rows.Scan(&x.KanbanLotID, &x.KanbanID, &x.MaterialCode, &x.MaterialName, &x.Unit, &x.Quantity); e != nil {
			return e
		}
		out.Scans = append(out.Scans, x)
	}
	return rows.Err()
}

func (s *Store) Scan(ctx context.Context, a Actor, sessionID uuid.UUID, value string) (Session, error) {
	kanban, err := normalizeKanban(value)
	if err != nil {
		return Session{}, err
	}
	err = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		var status string
		var dn uuid.UUID
		if e := tx.QueryRow(ctx, `SELECT status,delivery_note_id FROM receiving_sessions WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, a.TenantID, sessionID).Scan(&status, &dn); e != nil {
			return ErrNotFound
		}
		if status != SessionActive {
			return ErrConflict
		}
		var lot, lotDeliveryNote uuid.UUID
		var lotStatus string
		var alreadyScanned bool
		e := tx.QueryRow(ctx, `SELECT kl.id,kl.status,dnl.delivery_note_id,
 EXISTS(SELECT 1 FROM receiving_session_scans rss WHERE rss.tenant_id=kl.tenant_id AND rss.session_id=$3 AND rss.kanban_lot_id=kl.id)
 FROM kanban_lots kl
 JOIN delivery_note_lines dnl ON dnl.tenant_id=kl.tenant_id AND dnl.id=kl.delivery_note_line_id
 WHERE kl.tenant_id=$1 AND kl.kanban_id=$2
 FOR UPDATE OF kl`, a.TenantID, kanban, sessionID).Scan(&lot, &lotStatus, &lotDeliveryNote, &alreadyScanned)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrKanbanNotFound
		}
		if e != nil {
			return e
		}
		if lotDeliveryNote != dn {
			return ErrKanbanWrongDeliveryNote
		}
		if lotStatus != "ISSUED" {
			return ErrKanbanAlreadyReceived
		}
		if alreadyScanned {
			return ErrKanbanAlreadyScanned
		}
		_, e = tx.Exec(ctx, `INSERT INTO receiving_session_scans(tenant_id,session_id,kanban_lot_id,scanned_by_user_id) VALUES($1,$2,$3,$4)`, a.TenantID, sessionID, lot, a.UserID)
		if e != nil {
			return ErrKanbanAlreadyScanned
		}
		_, e = tx.Exec(ctx, `UPDATE receiving_sessions SET updated_at=now(),updated_by_user_id=$3,expires_at=now()+interval '30 minutes' WHERE tenant_id=$1 AND id=$2`, a.TenantID, sessionID, a.UserID)
		return e
	})
	if err != nil {
		return Session{}, err
	}
	return s.GetSession(ctx, a, sessionID)
}

func (s *Store) RemoveScan(ctx context.Context, a Actor, sessionID, lotID uuid.UUID) error {
	return database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		tag, e := tx.Exec(ctx, `DELETE FROM receiving_session_scans rss USING receiving_sessions rs WHERE rss.tenant_id=$1 AND rss.session_id=$2 AND rss.kanban_lot_id=$3 AND rs.tenant_id=rss.tenant_id AND rs.id=rss.session_id AND rs.status='ACTIVE'`, a.TenantID, sessionID, lotID)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}
func (s *Store) SetStatus(ctx context.Context, a Actor, id uuid.UUID, status, reason string) error {
	return database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		tag, e := tx.Exec(ctx, `UPDATE receiving_sessions SET status=$3,cancel_reason=$4,updated_by_user_id=$5,updated_at=now() WHERE tenant_id=$1 AND id=$2 AND status IN ('ACTIVE','PAUSED')`, a.TenantID, id, status, reason, a.UserID)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return ErrConflict
		}
		return nil
	})
}

func (s *Store) Complete(ctx context.Context, a Actor, id uuid.UUID) (Receiving, error) {
	var out Receiving
	err := database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		var dn, po uuid.UUID
		var number string
		var planned, previous int
		var status string
		if e := tx.QueryRow(ctx, `SELECT rs.delivery_note_id,dn.purchase_order_id,rs.receiving_number,rs.status,(SELECT COUNT(*) FROM delivery_note_lines d JOIN kanban_lots k ON k.tenant_id=d.tenant_id AND k.delivery_note_line_id=d.id WHERE d.tenant_id=rs.tenant_id AND d.delivery_note_id=rs.delivery_note_id),(SELECT COUNT(*) FROM delivery_note_lines d JOIN kanban_lots k ON k.tenant_id=d.tenant_id AND k.delivery_note_line_id=d.id WHERE d.tenant_id=rs.tenant_id AND d.delivery_note_id=rs.delivery_note_id AND k.status IN ('IN_STOCK','CONSUMED')) FROM receiving_sessions rs JOIN delivery_notes dn ON dn.tenant_id=rs.tenant_id AND dn.id=rs.delivery_note_id WHERE rs.tenant_id=$1 AND rs.id=$2 FOR UPDATE`, a.TenantID, id).Scan(&dn, &po, &number, &status, &planned, &previous); e != nil {
			return ErrNotFound
		}
		if status == "COMPLETED" {
			return ErrConflict
		}
		if status != "ACTIVE" {
			return ErrConflict
		}
		rows, e := tx.Query(ctx, `SELECT kl.id,kl.quantity,pol.raw_material_id,pol.base_unit_code_snapshot FROM receiving_session_scans rss JOIN kanban_lots kl ON kl.tenant_id=rss.tenant_id AND kl.id=rss.kanban_lot_id JOIN purchase_order_lines pol ON pol.tenant_id=kl.tenant_id AND pol.id=kl.purchase_order_line_id WHERE rss.tenant_id=$1 AND rss.session_id=$2 AND kl.status='ISSUED' FOR UPDATE OF kl`, a.TenantID, id)
		if e != nil {
			return e
		}
		type lot struct {
			id       uuid.UUID
			q        any
			material uuid.UUID
			unit     string
		}
		var lots []lot
		for rows.Next() {
			var l lot
			if e = rows.Scan(&l.id, &l.q, &l.material, &l.unit); e != nil {
				rows.Close()
				return e
			}
			lots = append(lots, l)
		}
		rows.Close()
		if len(lots) == 0 {
			return ErrValidation
		}
		rid := uuid.New()
		outstanding := planned - previous - len(lots)
		_, e = tx.Exec(ctx, `INSERT INTO receivings(id,tenant_id,session_id,delivery_note_id,purchase_order_id,receiving_number,receiving_date,planned_kanban,previously_received_kanban,received_now_kanban,outstanding_kanban,created_by_user_id,completed_by_user_id) VALUES($1,$2,$3,$4,$5,$6,current_date,$7,$8,$9,$10,$11,$11)`, rid, a.TenantID, id, dn, po, number, planned, previous, len(lots), outstanding, a.UserID)
		if e != nil {
			return e
		}
		for _, l := range lots {
			if _, e = tx.Exec(ctx, `UPDATE kanban_lots SET status='IN_STOCK',updated_by_user_id=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2 AND status='ISSUED'`, a.TenantID, l.id, a.UserID); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO receiving_kanban_lots(tenant_id,receiving_id,kanban_lot_id,quantity) SELECT $1,$2,id,quantity FROM kanban_lots WHERE tenant_id=$1 AND id=$3`, a.TenantID, rid, l.id); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO inventory_ledger_entries(tenant_id,event_type,kanban_lot_id,raw_material_id,quantity_delta,base_unit_code,reference_type,reference_id,created_by_user_id) SELECT $1,'RECEIVING',id,$3,quantity,$4,'RECEIVING',$2,$5 FROM kanban_lots WHERE tenant_id=$1 AND id=$6`, a.TenantID, rid, l.material, l.unit, a.UserID, l.id); e != nil {
				return e
			}
		}
		poStatus := "PARTIALLY_RECEIVED"
		dnStatus := "PARTIALLY_RECEIVED"
		if outstanding == 0 {
			poStatus = "FULLY_RECEIVED"
			dnStatus = "RECEIVED"
		}
		if _, e = tx.Exec(ctx, `UPDATE purchase_orders SET status=$3,updated_by_user_id=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, a.TenantID, po, poStatus, a.UserID); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE delivery_notes SET status=$3,updated_by_user_id=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, a.TenantID, dn, dnStatus, a.UserID); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO integration_outbox(tenant_id,event_type,aggregate_id,idempotency_key,payload) VALUES($1,'SAGE_GOODS_RECEIPT_CREATE',$2,$3,jsonb_build_object('receivingId',to_jsonb($2::uuid)))`, a.TenantID, rid, "receiving:"+rid.String()+":sage-goods-receipt"); e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `UPDATE receiving_sessions SET status='COMPLETED',updated_by_user_id=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, a.UserID)
		out.ID = rid
		return e
	})
	if err != nil {
		return Receiving{}, err
	}
	return s.GetReceiving(ctx, a, out.ID)
}

func (s *Store) GetReceiving(ctx context.Context, a Actor, id uuid.UUID) (Receiving, error) {
	var o Receiving
	err := database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		e := tx.QueryRow(ctx, `SELECT r.id,r.receiving_number,dn.delivery_note_number,p.po_number,s.name,r.status,r.sage_receipt_number,r.receiving_date,r.planned_kanban,r.previously_received_kanban,r.received_now_kanban,r.outstanding_kanban,u.display_name FROM receivings r JOIN delivery_notes dn ON dn.tenant_id=r.tenant_id AND dn.id=r.delivery_note_id JOIN purchase_orders p ON p.tenant_id=r.tenant_id AND p.id=r.purchase_order_id JOIN suppliers s ON s.tenant_id=p.tenant_id AND s.id=p.supplier_id JOIN users u ON u.tenant_id=r.tenant_id AND u.id=r.completed_by_user_id WHERE r.tenant_id=$1 AND r.id=$2`, a.TenantID, id).Scan(&o.ID, &o.ReceivingNumber, &o.DeliveryNoteNumber, &o.PONumber, &o.SupplierName, &o.Status, &o.SageReceiptNumber, &o.ReceivingDate, &o.Planned, &o.PreviouslyReceived, &o.ReceivedNow, &o.Outstanding, &o.CreatedBy)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		rows, e := tx.Query(ctx, `SELECT kl.id,kl.kanban_id,pol.raw_material_code_snapshot,pol.raw_material_name_snapshot,pol.base_unit_code_snapshot,kl.quantity FROM receiving_kanban_lots rkl JOIN kanban_lots kl ON kl.tenant_id=rkl.tenant_id AND kl.id=rkl.kanban_lot_id JOIN purchase_order_lines pol ON pol.tenant_id=kl.tenant_id AND pol.id=kl.purchase_order_line_id WHERE rkl.tenant_id=$1 AND rkl.receiving_id=$2 ORDER BY kl.kanban_id`, a.TenantID, id)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var scan Scan
			if e = rows.Scan(&scan.KanbanLotID, &scan.KanbanID, &scan.MaterialCode, &scan.MaterialName, &scan.Unit, &scan.Quantity); e != nil {
				return e
			}
			o.Scans = append(o.Scans, scan)
		}
		return rows.Err()
	})
	return o, err
}
func (s *Store) List(ctx context.Context, a Actor, query ListQuery) (items []Receiving, err error) {
	var createdFrom, createdTo any
	if !query.CreatedFrom.IsZero() {
		createdFrom = query.CreatedFrom
	}
	if !query.CreatedToExclusive.IsZero() {
		createdTo = query.CreatedToExclusive
	}
	err = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT r.id,p.supplier_id,r.receiving_number,dn.delivery_note_number,p.po_number,s.name,r.status,r.sage_receipt_number,r.receiving_date,r.planned_kanban,r.previously_received_kanban,r.received_now_kanban,r.outstanding_kanban,u.display_name,r.created_at FROM receivings r JOIN delivery_notes dn ON dn.tenant_id=r.tenant_id AND dn.id=r.delivery_note_id JOIN purchase_orders p ON p.tenant_id=r.tenant_id AND p.id=r.purchase_order_id JOIN suppliers s ON s.tenant_id=p.tenant_id AND s.id=p.supplier_id JOIN users u ON u.tenant_id=r.tenant_id AND u.id=r.completed_by_user_id WHERE r.tenant_id=$1 AND ($2::uuid='00000000-0000-0000-0000-000000000000' OR p.supplier_id=$2) AND ($3::timestamptz IS NULL OR r.created_at >= $3) AND ($4::timestamptz IS NULL OR r.created_at < $4) ORDER BY r.created_at DESC,r.id DESC`, a.TenantID, query.SupplierID, createdFrom, createdTo)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var o Receiving
			if e = rows.Scan(&o.ID, &o.SupplierID, &o.ReceivingNumber, &o.DeliveryNoteNumber, &o.PONumber, &o.SupplierName, &o.Status, &o.SageReceiptNumber, &o.ReceivingDate, &o.Planned, &o.PreviouslyReceived, &o.ReceivedNow, &o.Outstanding, &o.CreatedBy, &o.CreatedAt); e != nil {
				return e
			}
			items = append(items, o)
		}
		return rows.Err()
	})
	return
}

var _ = fmt.Sprintf
