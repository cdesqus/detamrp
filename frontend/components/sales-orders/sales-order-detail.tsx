'use client';

import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { MaterialRequirements, MaterialRequirement, RequirementResult } from './material-requirements';
import { formatMoney, formatQuantity } from '../../lib/number-format';
import { useToast } from '../toast/toast-provider';
import { Icon } from '../icons';

type Line = { id:string; itemCode:string; name:string; unit:string; quantity:string; deliveredQuantity:string; remainingQuantity:string; salesPrice:string; currency:string };
type Order = { id:string; number:string; customerName:string; status:string; orderDate:string; lines:Line[] };
type Calculation = { lineId:string; result:RequirementResult };

function combineRequirements(calculations:Calculation[]):RequirementResult {
  const materials = new Map<string, MaterialRequirement>();
  const totals: Record<string, number> = {};
  for (const calculation of calculations) {
    for (const material of calculation.result.materials) {
      const current = materials.get(material.itemId);
      if (current) {
        current.quantity = String(Number(current.quantity) + Number(material.quantity));
        current.kanbanEquivalent = String(Number(current.kanbanEquivalent) + Number(material.kanbanEquivalent));
        current.purchaseKanban = String(Number(current.purchaseKanban) + Number(material.purchaseKanban));
        if (material.value !== undefined) current.value = String(Number(current.value ?? 0) + Number(material.value));
      } else materials.set(material.itemId, { ...material });
    }
    for (const [currency, total] of Object.entries(calculation.result.totals ?? {})) totals[currency] = (totals[currency] ?? 0) + Number(total);
  }
  return { nodes: [], materials: [...materials.values()], totals: Object.fromEntries(Object.entries(totals).map(([currency, total]) => [currency, String(total)])), costComplete: calculations.every(item => item.result.costComplete !== false) };
}

export function SalesOrderDetail({ id }: { id:string }) {
  const router = useRouter();
  const { showSuccess } = useToast();
  const [order, setOrder] = useState<Order | null>(null);
  const [calculation, setCalculation] = useState<Calculation[]>([]);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    fetch(`/api/sales-orders/${id}`, { credentials: 'include' })
      .then(response => response.ok ? response.json() : Promise.reject())
      .then((item:Order) => {
        setOrder(item);
        if (item.status === 'SUBMITTED') return fetch(`/api/sales-orders/${id}/requirements`, { credentials: 'include' })
          .then(response => response.ok ? response.json() : Promise.reject())
          .then(data => setCalculation(data.lines ?? []));
      })
      .catch(() => setError('Sales order could not be loaded'));
  }, [id]);

  const combined = useMemo(() => combineRequirements(calculation), [calculation]);
  const fgTotals = useMemo(() => order?.lines.reduce<Record<string, number>>((totals, line) => { totals[line.currency] = (totals[line.currency] ?? 0) + Number(line.quantity) * Number(line.salesPrice); return totals; }, {}) ?? {}, [order]);

  async function submit() {
    setSubmitting(true); setError('');
    try {
      const response = await fetch(`/api/sales-orders/${id}/submit`, { method: 'POST', credentials: 'include' });
      if (!response.ok) { const body = await response.json(); setError(body.message ?? 'Sales order could not be submitted'); return; }
      const item = await response.json(); setOrder(item);
      showSuccess('Sales order submitted successfully. BOM requirements are now available.');
      const requirements = await fetch(`/api/sales-orders/${id}/requirements`, { credentials: 'include' });
      if (requirements.ok) { const data = await requirements.json(); setCalculation(data.lines ?? []); }
    } catch { setError('Sales order could not be submitted'); } finally { setSubmitting(false); }
  }

  async function remove() {
    if (!window.confirm('Delete this draft sales order? This cannot be undone.')) return;
    setError('');
    try {
      const response = await fetch(`/api/sales-orders/${id}`, { method: 'DELETE', credentials: 'include' });
      if (!response.ok) { const body = await response.json().catch(() => ({})); setError(body.message ?? 'Sales order could not be deleted'); return; }
      showSuccess('Sales order deleted successfully.'); router.push('/sales-orders');
    } catch { setError('Sales order could not be deleted'); }
  }\n  if (error && !order) return <section className="module-index"><div className="table-empty"><strong>Could not load sales order</strong><span>{error}</span></div></section>;
  if (!order) return <section className="module-index"><p className="muted">Loading sales order...</p></section>;

  return <section className="module-index">
    <div className="page-title-row"><div><h1>{order.number}</h1><p className="muted">{order.customerName} · {order.status}</p></div><div className="toolbar-actions"><button onClick={() => router.push('/sales-orders')}>Back to orders</button>{order.status === 'DRAFT' && <><button className="primary-button" disabled={submitting} onClick={submit}>{submitting ? 'Submitting...' : 'Submit sales order'}</button><button className="table-action" onClick={remove}>Delete draft</button></>}{order.status === 'SUBMITTED' && <button className="primary-button" onClick={() => router.push(`/sales-orders/${id}/delivery`)}>Create delivery</button>}</div></div>
    {error && <p className="form-error" role="alert">{error}</p>}
    <div className="table-frame"><table><thead><tr><th>Finished Good</th><th>Ordered</th><th>Delivered</th><th>Remaining</th><th>Unit</th><th>Sales Price</th><th>Total Price</th><th>Currency</th></tr></thead><tbody>{order.lines.map(line => <tr key={line.id}><td>{line.itemCode} — {line.name}</td><td>{formatQuantity(line.quantity)}</td><td>{formatQuantity(line.deliveredQuantity)}</td><td>{formatQuantity(line.remainingQuantity)}</td><td>{line.unit}</td><td>{formatMoney(line.salesPrice, line.currency)}</td><td>{formatMoney(String(Number(line.quantity) * Number(line.salesPrice)), line.currency)}</td><td>{line.currency}</td></tr>)}</tbody></table></div>
    <div className="toolbar-actions" style={{ justifyContent:'flex-end', gap:24, marginTop:12 }}>{Object.entries(fgTotals).map(([currency,total]) => <strong key={currency}>Total FG Value ({currency}): {formatMoney(String(total), currency)}</strong>)}</div>
    {order.status === 'DRAFT' ? <><div className="page-title-row"><div><h2>Material Requirements</h2><p className="muted">Submit this order to freeze its BOM calculation and enable exports.</p></div></div></> : <><div className="page-title-row"><div><h2>Finished Goods</h2><p className="muted">Open a finished good to review its BOM breakdown.</p></div></div><div className="table-frame"><table><thead><tr><th>Finished Good</th><th>Ordered</th><th>BOM</th><th>Status</th></tr></thead><tbody>{order.lines.map((line, index) => { const result = calculation[index]?.result; const root = result?.nodes?.[0]; const isOpen = expanded.has(line.id); return <tr key={line.id}><td colSpan={4}><button className="table-action" style={{ width:'100%', display:'grid', gridTemplateColumns:'24px 1fr auto auto', alignItems:'center', textAlign:'left', gap:12 }} onClick={() => setExpanded(value => { const next = new Set(value); if (next.has(line.id)) next.delete(line.id); else next.add(line.id); return next; })} aria-expanded={isOpen}><Icon name={isOpen ? 'chevron-down' : 'chevron-right'} size={16}/><span><strong>{line.itemCode} — {line.name}</strong><br/><small>{formatQuantity(line.quantity)} {line.unit}</small></span><span>{root ? 'BOM Rev 1' : 'BOM'}</span><span>{root ? 'ACTIVE' : '—'}</span></button>{isOpen && <div style={{ padding:'12px 24px 12px 48px' }}>{result?.nodes?.filter(node => node.depth === 1).map(node => <div key={node.path} style={{ display:'flex', justifyContent:'space-between', padding:'6px 0' }}><span>{node.itemCode} — {node.name}</span><span>{formatQuantity(node.quantity)} {node.unit}</span></div>)}</div>}</td></tr>; })}</tbody></table></div><div className="page-title-row"><div><h2>Material Requirements</h2><p className="muted">Combined requirements across this submitted order.</p></div></div><MaterialRequirements result={combined} showCosts showBreakdown={false}/></>}
  </section>;
}




