'use client';

import { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';

type Order = { id: string; poNumber: string; supplierId: string; supplierName?: string; orderDate: string; expectedDeliveryDate: string; status: string; totalAmount: string; currency: string; createdBy?: { displayName?: string } };
type Supplier = { id: string; code: string; name: string };

function date(value: string) { return value ? value.slice(0, 10) : '—'; }

export function SupplierOrderIndex() {
  const router = useRouter();
  const [items, setItems] = useState<Order[]>([]);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const load = useCallback(async () => {
    setLoading(true); setError('');
    try {
      const [response, supplierResponse] = await Promise.all([fetch(`/api/purchase-orders?${new URLSearchParams({ search })}`, { credentials: 'include' }), fetch('/api/master-data/suppliers?active=true&limit=200', { credentials: 'include' })]);
      if (!response.ok) throw new Error('Supplier orders could not be loaded');
      const [data, supplierData] = await Promise.all([response.json() as Promise<{ items: Order[]; total: number }>, supplierResponse.ok ? supplierResponse.json() as Promise<{ items: Supplier[] }> : Promise.resolve({ items: [] as Supplier[] })]);
      const names = new Map((supplierData.items ?? []).map(supplier => [supplier.id, `${supplier.code} — ${supplier.name}`]));
      setItems((data.items ?? []).map(order => ({ ...order, supplierName: names.get(order.supplierId) ?? '—' }))); setTotal(data.total ?? 0);
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Supplier orders could not be loaded'); }
    finally { setLoading(false); }
  }, [search]);
  useEffect(() => { const timer = setTimeout(load, 200); return () => clearTimeout(timer); }, [load]);
  return <section className="module-index supplier-order-index">
    <div className="page-title-row"><div><h1>Supplier Orders</h1><p className="muted">Create, save, and submit purchase orders.</p></div><button className="primary-button" onClick={() => router.push('/supplier-orders/new')}>Create order</button></div>
    <div className="table-toolbar"><input aria-label="Search Supplier Orders" type="search" value={search} onChange={event => setSearch(event.target.value)} placeholder="Search PO" /><div className="toolbar-actions"><span>{total} records</span></div></div>
    <div className="table-frame"><table><thead><tr><th>PO Number</th><th>Supplier</th><th>Order Date</th><th>Expected Date</th><th>Status</th><th>Total</th><th>Created By</th><th>Action</th></tr></thead><tbody>
      {loading ? <tr><td colSpan={8}><div className="table-empty">Loading...</div></td></tr> : error ? <tr><td colSpan={8}><div className="table-empty"><strong>Could not load supplier orders</strong><span>{error}</span><button className="table-action" onClick={load}>Retry</button></div></td></tr> : items.length === 0 ? <tr><td colSpan={8}><div className="table-empty"><strong>No supplier orders yet</strong><span>Create order to start.</span></div></td></tr> : items.map(order => <tr key={order.id}><td>{order.poNumber}</td><td>{order.supplierName ?? '—'}</td><td>{date(order.orderDate)}</td><td>{date(order.expectedDeliveryDate)}</td><td><span className="status-pill">{order.status.replaceAll('_', ' ')}</span></td><td>{order.currency} {formatDecimal(order.totalAmount)}</td><td>{order.createdBy?.displayName ?? '—'}</td><td><button className="table-action" onClick={() => router.push(`/supplier-orders/${order.id}`)}>Open</button></td></tr>)}
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
