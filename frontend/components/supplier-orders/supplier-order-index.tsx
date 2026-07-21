'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useCurrentUser } from '../app-shell/app-shell';

type Order = { id: string; poNumber: string; supplierId: string; supplierName?: string; orderDate: string; expectedDeliveryDate: string; status: string; totalAmount?: string; currency: string; createdBy?: { displayName?: string } };
type Props = { permissions?: string[] };

function date(value: string) { return value ? value.slice(0, 10) : '—'; }

export function SupplierOrderIndex({ permissions }: Props = {}) {
  const router = useRouter();
  const currentUser = useCurrentUser();
  const canViewPrices = (permissions ?? currentUser?.permissions ?? []).includes('po.price.view');
  const [items, setItems] = useState<Order[]>([]);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const requestSequence = useRef(0);
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
  const columnCount = canViewPrices ? 8 : 7;
  return <section className="module-index supplier-order-index">
    <div className="page-title-row"><div><h1>Supplier Orders</h1><p className="muted">Create, save, and submit purchase orders.</p></div><button className="primary-button" onClick={() => router.push('/supplier-orders/new')}>Create order</button></div>
    <div className="table-toolbar"><input aria-label="Search Supplier Orders" type="search" value={search} onChange={event => setSearch(event.target.value)} placeholder="Search PO" /><div className="toolbar-actions"><span>{total} records</span></div></div>
    <div className="table-frame"><table><thead><tr><th>PO Number</th><th>Supplier</th><th>Order Date</th><th>Expected Date</th><th>Status</th>{canViewPrices && <th>Total</th>}<th>Created By</th><th>Action</th></tr></thead><tbody>
      {loading ? <tr><td colSpan={columnCount}><div className="table-empty" role="status">Loading...</div></td></tr> : error ? <tr><td colSpan={columnCount}><div className="table-empty" role="alert"><strong>Could not load supplier orders</strong><span>{error}</span><button className="table-action" onClick={load}>Retry</button></div></td></tr> : items.length === 0 ? <tr><td colSpan={columnCount}><div className="table-empty" role="status"><strong>No supplier orders yet</strong><span>Create order to start.</span></div></td></tr> : items.map(order => <tr key={order.id}><td>{order.poNumber}</td><td>{order.supplierName ?? '—'}</td><td>{date(order.orderDate)}</td><td>{date(order.expectedDeliveryDate)}</td><td><span className="status-pill">{order.status.replaceAll('_', ' ')}</span></td>{canViewPrices && <td>{order.currency} {formatDecimal(order.totalAmount)}</td>}<td>{order.createdBy?.displayName ?? '—'}</td><td><button className="table-action" aria-label={`Open ${order.poNumber}`} onClick={() => router.push(`/supplier-orders/${order.id}`)}>Open</button></td></tr>)}
    </tbody></table></div>
  </section>;
}

export function formatDecimal(value: string | number | undefined) {
  const raw = String(value ?? '0').trim();
  const negative = raw.startsWith('-'); const digits = (negative ? raw.slice(1) : raw).split('.');
  const whole = (digits[0] || '0').replace(/^0+(?=\d)/, '') || '0';
  const fraction = (digits[1] || '').replace(/\D/g, '').padEnd(6, '0').slice(0, 6);
  return `${negative ? '-' : ''}${whole}.${fraction}`;
}
