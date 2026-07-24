'use client';

import { KeyboardEvent as ReactKeyboardEvent, useCallback, useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useCurrentUser } from '../app-shell/app-shell';
import { OrderStatusBadge } from './order-status';
import { formatMoney } from '../../lib/number-format';
import { Icon } from '../icons';
import { pagedItems, TablePagination } from '../table-pagination';

type Order = { id: string; poNumber: string; supplierId: string; supplierName?: string; orderDate: string; expectedDeliveryDate: string; status: string; totalAmount?: string; currency: string; sagePurchaseOrderNumber?: string; createdBy?: { displayName?: string }; documents?: { deliveryNoteId: string; deliveryNoteNumber: string; kanbanCount: number; issuedAt: string } | null };
type Props = { permissions?: string[] };
type DocumentLinkProps = { href: string; label: string };
const maxKanbanLabelsPerPDF = 1_000;

function date(value: string) { return value ? value.slice(0, 10) : '—'; }
function DocumentLink({ href, label }: DocumentLinkProps) {
  return <a className="icon-action" title={label} href={href} target="_blank" rel="noopener noreferrer" aria-label={label}><Icon name="pdf" size={14} /></a>;
}
export function SupplierOrderIndex({ permissions }: Props = {}) {
  const router = useRouter();
  const currentUser = useCurrentUser();
  const permissionList = permissions ?? currentUser?.permissions ?? [];
  const canViewPrices = permissionList.includes('po.price.view');
  const canEditDraft = permissionList.includes('po.edit_draft');
  const canSubmit = permissionList.includes('po.submit');
  const [items, setItems] = useState<Order[]>([]);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [menuOrderId, setMenuOrderId] = useState('');
  const [mailOrderId, setMailOrderId] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [cancellingOrder, setCancellingOrder] = useState<Order | null>(null);
  const [cancelling, setCancelling] = useState(false);
  const [cancelError, setCancelError] = useState('');
  const requestSequence = useRef(0);
  const actionTriggers = useRef<Record<string, HTMLButtonElement | null>>({});
  const openTriggers = useRef<Record<string, HTMLButtonElement | null>>({});
  const dialogRef = useRef<HTMLDivElement | null>(null);

  const load = useCallback(async () => {
    const request = ++requestSequence.current;
    setLoading(true); setError('');
    try {
      const response = await fetch(`/api/purchase-orders?${new URLSearchParams({ search })}`, { credentials: 'include' });
      if (!response.ok) throw new Error('Supplier orders could not be loaded');
      const data = await response.json() as { items: Order[]; total: number };
      if (request !== requestSequence.current) return;
      setItems(data.items ?? []); setTotal(data.total ?? 0); setPage(1);
    } catch (cause) { if (request === requestSequence.current) setError(cause instanceof Error ? cause.message : 'Supplier orders could not be loaded'); }
    finally { if (request === requestSequence.current) setLoading(false); }
  }, [search]);
  useEffect(() => { const timer = setTimeout(load, 200); return () => clearTimeout(timer); }, [load]);

  const closeCancellation = useCallback(() => {
    if (cancelling) return;
    const trigger = cancellingOrder ? actionTriggers.current[cancellingOrder.id] : null;
    setCancellingOrder(null); setCancelError('');
    trigger?.focus();
  }, [cancelling, cancellingOrder]);

  useEffect(() => {
    if (!cancellingOrder) return;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !cancelling) { event.preventDefault(); closeCancellation(); }
    };
    document.addEventListener('keydown', closeOnEscape);
    return () => document.removeEventListener('keydown', closeOnEscape);
  }, [cancelling, cancellingOrder, closeCancellation]);
  useEffect(() => { if (cancelling) dialogRef.current?.focus(); }, [cancelling]);

  const openCancellation = (order: Order) => {
    setMenuOrderId(''); setCancelError(''); setCancellingOrder(order);
  };

  const containDialogFocus = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (event.key !== 'Tab') return;
    const controls = Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('button:not(:disabled)'));
    if (controls.length === 0) { event.preventDefault(); event.currentTarget.focus(); return; }
    const first = controls[0]; const last = controls[controls.length - 1];
    if (document.activeElement === event.currentTarget) { event.preventDefault(); (event.shiftKey ? last : first)?.focus(); }
    else if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus(); }
    else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
  };

  async function cancelDraft() {
    if (!cancellingOrder || cancelling) return;
    setCancelling(true); setCancelError('');
    try {
      const response = await fetch(`/api/purchase-orders/${cancellingOrder.id}/cancel`, { method: 'POST', credentials: 'include' });
      if (!response.ok) {
        const body = await response.json() as { message?: string };
        throw new Error(body.message ?? 'Supplier order could not be cancelled');
      }
      const orderId = cancellingOrder.id;
      await load();
      setCancellingOrder(null);
      window.setTimeout(() => (actionTriggers.current[orderId] ?? openTriggers.current[orderId])?.focus(), 0);
    } catch (cause) {
      setCancelError(cause instanceof Error ? cause.message : 'Supplier order could not be cancelled');
    } finally {
      setCancelling(false);
    }
  }

  async function sendApproval(order: Order) {
    const response = await fetch(`/api/purchase-orders/${order.id}/submit`, { method: 'POST', credentials: 'include' });
    if (!response.ok) { setError('Supplier order could not be submitted'); return; }
    setMailOrderId('');
    await load();
  }

  const columnCount = canViewPrices ? 11 : 10;
  return <section className="module-index supplier-order-index">
    <div className="page-title-row"><div><h1>Supplier Orders</h1><p className="muted">Create, save, and submit purchase orders.</p></div><button className="primary-button" onClick={() => router.push('/supplier-orders/new')}>Create order</button></div>
    <div className="table-toolbar"><input aria-label="Search Supplier Orders" type="search" value={search} onChange={event => setSearch(event.target.value)} placeholder="Search PO" /><div className="toolbar-actions"><span>{total} records</span></div></div>
    <div className="table-frame"><table><thead><tr><th className="transaction-number">No.</th><th>Actions</th><th>Documents</th><th>Status</th><th>PO Number</th><th>Supplier</th><th>Order Date</th><th>Expected Date</th>{canViewPrices && <th>Total</th>}<th>Sage No.</th><th>Created By</th></tr></thead><tbody>
      {loading ? <tr><td colSpan={columnCount}><div className="table-empty" role="status">Loading...</div></td></tr> : error ? <tr><td colSpan={columnCount}><div className="table-empty" role="alert"><strong>Could not load supplier orders</strong><span>{error}</span><button className="table-action" onClick={load}>Retry</button></div></td></tr> : items.length === 0 ? <tr><td colSpan={columnCount}><div className="table-empty" role="status"><strong>No supplier orders yet</strong><span>Create order to start.</span></div></td></tr> : pagedItems(items, page, pageSize).map((order, index) => {
        const documentsAvailable = ['APPROVED', 'PARTIALLY_RECEIVED', 'FULLY_RECEIVED'].includes(order.status) && order.documents;
        const labelsAvailable = documentsAvailable ? documentsAvailable.kanbanCount <= maxKanbanLabelsPerPDF : false;
        return <tr key={order.id}><td className="transaction-number">{(page - 1) * pageSize + index + 1}</td><td><div className="table-icon-group"><button ref={element => { openTriggers.current[order.id] = element; }} className="icon-action" title={`Open ${order.poNumber}`} aria-label={`Open ${order.poNumber}`} onClick={() => router.push(`/supplier-orders/${order.id}`)}><Icon name="open" size={14} /></button><button className="icon-action" title={`Edit ${order.poNumber}`} aria-label={`Edit ${order.poNumber}`} disabled={!canEditDraft || order.status !== 'DRAFT'} onClick={() => router.push(`/supplier-orders/${order.id}`)}><Icon name="edit" size={14} /></button><div className="supplier-order-row-menu"><button className="icon-action" title={`Message actions for ${order.poNumber}`} aria-label={`Message actions for ${order.poNumber}`} onClick={() => setMailOrderId(value => value === order.id ? '' : order.id)}><Icon name="mail" size={14} /></button>{mailOrderId === order.id && <div className="supplier-order-row-menu-popover"><button disabled={!canSubmit || order.status !== 'DRAFT'} onClick={() => void sendApproval(order)}>Send to Approval</button><button disabled title="SMTP is not configured">Send to Supplier</button></div>}</div>{canEditDraft && order.status === 'DRAFT' && <div className="supplier-order-row-menu"><button ref={element => { actionTriggers.current[order.id] = element; }} className="icon-action" title={`Cancel ${order.poNumber}`} aria-label={`Actions for ${order.poNumber}`} onClick={() => setMenuOrderId(value => value === order.id ? '' : order.id)}><Icon name="cancel" size={14} /></button>{menuOrderId === order.id && <div id={`draft-actions-${order.id}`} data-testid={`draft-actions-${order.id}`} className="supplier-order-row-menu-popover supplier-order-row-menu-popover--upward"><button onClick={() => openCancellation(order)}>Cancel Draft</button></div>}</div>}</div></td><td><div className="table-icon-group"><DocumentLink href={`/api/purchase-orders/${order.id}/documents/po.pdf`} label={`Open PO PDF for ${order.poNumber}`} />{documentsAvailable && <DocumentLink href={`/api/purchase-orders/${order.id}/documents/delivery-note.pdf`} label={`Open DN PDF for ${order.poNumber}`} />}{labelsAvailable && <DocumentLink href={`/api/purchase-orders/${order.id}/documents/kanban-labels.pdf`} label={`Open Kanban labels PDF for ${order.poNumber}`} />}</div></td><td><OrderStatusBadge status={order.status} /></td><td>{order.poNumber}</td><td>{order.supplierName ?? '—'}</td><td>{date(order.orderDate)}</td><td>{date(order.expectedDeliveryDate)}</td>{canViewPrices && <td>{formatMoney(order.totalAmount, order.currency)}</td>}<td>{order.sagePurchaseOrderNumber || '—'}</td><td>{order.createdBy?.displayName ?? '—'}</td></tr>;
      })}
    </tbody></table><TablePagination page={page} pageSize={pageSize} total={items.length} onPageChange={setPage} onPageSizeChange={size => { setPageSize(size); setPage(1); }} /></div>
    {cancellingOrder ? <><button tabIndex={-1} className="crud-scrim" aria-label="Close cancel draft confirmation" onClick={closeCancellation} disabled={cancelling} /><div ref={dialogRef} tabIndex={-1} className="crud-modal crud-modal--compact" role="dialog" aria-modal="true" aria-labelledby="cancel-draft-title" aria-describedby="cancel-draft-description" onKeyDown={containDialogFocus}><div className="crud-modal-heading"><div><strong id="cancel-draft-title">Cancel draft {cancellingOrder.poNumber}</strong><span id="cancel-draft-description">This action cannot be undone.</span></div><button aria-label="Close cancel draft confirmation" onClick={closeCancellation} disabled={cancelling}>×</button></div><form onSubmit={event => { event.preventDefault(); void cancelDraft(); }}>{cancelError && <p className="form-error" role="alert">{cancelError}</p>}<div className="crud-fields"><p>Cancel this draft purchase order?</p></div><div className="crud-actions"><button type="button" onClick={closeCancellation} disabled={cancelling}>Keep draft</button><button autoFocus className="primary-button" disabled={cancelling}>{cancelling ? 'Cancelling...' : 'Confirm cancellation'}</button></div></form></div></> : null}
  </section>;
}
