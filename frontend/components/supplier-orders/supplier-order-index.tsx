'use client';

import { KeyboardEvent as ReactKeyboardEvent, useCallback, useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useCurrentUser } from '../app-shell/app-shell';

type Order = { id: string; poNumber: string; supplierId: string; supplierName?: string; orderDate: string; expectedDeliveryDate: string; status: string; totalAmount?: string; currency: string; createdBy?: { displayName?: string } };
type Props = { permissions?: string[] };

function date(value: string) { return value ? value.slice(0, 10) : '—'; }

export function SupplierOrderIndex({ permissions }: Props = {}) {
  const router = useRouter();
  const currentUser = useCurrentUser();
  const permissionList = permissions ?? currentUser?.permissions ?? [];
  const canViewPrices = permissionList.includes('po.price.view');
  const canEditDraft = permissionList.includes('po.edit_draft');
  const [items, setItems] = useState<Order[]>([]);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [menuOrderId, setMenuOrderId] = useState('');
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
      setItems(data.items ?? []); setTotal(data.total ?? 0);
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

  const columnCount = canViewPrices ? 8 : 7;
  return <section className="module-index supplier-order-index">
    <div className="page-title-row"><div><h1>Supplier Orders</h1><p className="muted">Create, save, and submit purchase orders.</p></div><button className="primary-button" onClick={() => router.push('/supplier-orders/new')}>Create order</button></div>
    <div className="table-toolbar"><input aria-label="Search Supplier Orders" type="search" value={search} onChange={event => setSearch(event.target.value)} placeholder="Search PO" /><div className="toolbar-actions"><span>{total} records</span></div></div>
    <div className="table-frame"><table><thead><tr><th>PO Number</th><th>Supplier</th><th>Order Date</th><th>Expected Date</th><th>Status</th>{canViewPrices && <th>Total</th>}<th>Created By</th><th>Action</th></tr></thead><tbody>
      {loading ? <tr><td colSpan={columnCount}><div className="table-empty" role="status">Loading...</div></td></tr> : error ? <tr><td colSpan={columnCount}><div className="table-empty" role="alert"><strong>Could not load supplier orders</strong><span>{error}</span><button className="table-action" onClick={load}>Retry</button></div></td></tr> : items.length === 0 ? <tr><td colSpan={columnCount}><div className="table-empty" role="status"><strong>No supplier orders yet</strong><span>Create order to start.</span></div></td></tr> : items.map(order => <tr key={order.id}><td>{order.poNumber}</td><td>{order.supplierName ?? '—'}</td><td>{date(order.orderDate)}</td><td>{date(order.expectedDeliveryDate)}</td><td><span className="status-pill">{order.status.replaceAll('_', ' ')}</span></td>{canViewPrices && <td>{order.currency} {formatDecimal(order.totalAmount)}</td>}<td>{order.createdBy?.displayName ?? '—'}</td><td><div className="supplier-order-row-actions"><button ref={element => { openTriggers.current[order.id] = element; }} className="table-action" aria-label={`Open ${order.poNumber}`} onClick={() => router.push(`/supplier-orders/${order.id}`)}>Open</button>{canEditDraft && order.status === 'DRAFT' && <div className="supplier-order-row-menu"><button ref={element => { actionTriggers.current[order.id] = element; }} className="supplier-order-row-menu-trigger" aria-label={`Actions for ${order.poNumber}`} aria-haspopup="menu" aria-expanded={menuOrderId === order.id} onClick={() => setMenuOrderId(current => current === order.id ? '' : order.id)}>...</button>{menuOrderId === order.id && <div className="supplier-order-row-menu-popover supplier-order-row-menu-popover--upward" role="menu"><button role="menuitem" onClick={() => openCancellation(order)}>Cancel Draft</button></div>}</div>}</div></td></tr>)}
    </tbody></table></div>
    {cancellingOrder ? <><button tabIndex={-1} className="crud-scrim" aria-label="Close cancel draft confirmation" onClick={closeCancellation} disabled={cancelling} /><div ref={dialogRef} tabIndex={-1} className="crud-modal crud-modal--compact" role="dialog" aria-modal="true" aria-labelledby="cancel-draft-title" aria-describedby="cancel-draft-description" onKeyDown={containDialogFocus}><div className="crud-modal-heading"><div><strong id="cancel-draft-title">Cancel draft {cancellingOrder.poNumber}</strong><span id="cancel-draft-description">This action cannot be undone.</span></div><button aria-label="Close cancel draft confirmation" onClick={closeCancellation} disabled={cancelling}>×</button></div><form onSubmit={event => { event.preventDefault(); void cancelDraft(); }}>{cancelError && <p className="form-error" role="alert">{cancelError}</p>}<div className="crud-fields"><p>Cancel this draft purchase order?</p></div><div className="crud-actions"><button type="button" onClick={closeCancellation} disabled={cancelling}>Keep draft</button><button autoFocus className="primary-button" disabled={cancelling}>{cancelling ? 'Cancelling...' : 'Confirm cancellation'}</button></div></form></div></> : null}
  </section>;
}

export function formatDecimal(value: string | number | undefined) {
  const raw = String(value ?? '0').trim();
  const negative = raw.startsWith('-'); const digits = (negative ? raw.slice(1) : raw).split('.');
  const whole = (digits[0] || '0').replace(/^0+(?=\d)/, '') || '0';
  const fraction = (digits[1] || '').replace(/\D/g, '').padEnd(6, '0').slice(0, 6);
  return `${negative ? '-' : ''}${whole}.${fraction}`;
}
