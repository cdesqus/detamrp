'use client';

import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { DashboardChart } from './dashboard-chart';
import { DashboardSnapshot, defaultDashboardDates, emptyDashboardSnapshot } from './dashboard-data';
import { StatusDonut } from './status-donut';
import { SupplierBars } from './supplier-bars';
import { TrendChart } from './trend-chart';

type Supplier = { id: string; code: string; name: string };

export function DashboardView() {
  const router = useRouter(), searchParams = useSearchParams();
  const defaults = useMemo(() => defaultDashboardDates(), []);
  const [from, setFrom] = useState(searchParams.get('from') || defaults.from);
  const [to, setTo] = useState(searchParams.get('to') || defaults.to);
  const [supplierID, setSupplierID] = useState(searchParams.get('supplierId') || '');
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [data, setData] = useState<DashboardSnapshot>(emptyDashboardSnapshot);
  const [loading, setLoading] = useState(true), [error, setError] = useState('');
  const request = useRef(0);

  const load = useCallback(async () => {
    const sequence = ++request.current;
    setLoading(true); setError('');
    const query = searchParams.toString();
    try {
      const response = await fetch(`/api/dashboard${query ? `?${query}` : ''}`, { credentials: 'include' });
      if (!response.ok) throw new Error();
      const snapshot = await response.json() as DashboardSnapshot;
      if (sequence === request.current) setData(snapshot);
    } catch {
      if (sequence === request.current) setError('Dashboard could not be loaded.');
    } finally {
      if (sequence === request.current) setLoading(false);
    }
  }, [searchParams]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    const controller = new AbortController();
    fetch('/api/master-data/suppliers?active=true&limit=200', { credentials: 'include', signal: controller.signal })
      .then(response => response.ok ? response.json() : Promise.reject())
      .then(payload => setSuppliers(payload.items ?? [])).catch(() => {});
    return () => controller.abort();
  }, []);

  function apply(event: FormEvent) {
    event.preventDefault();
    if (from > to) { setError('From date must be on or before To date.'); return; }
    const query = new URLSearchParams({ from, to });
    if (supplierID) query.set('supplierId', supplierID);
    router.replace(`/dashboard?${query}`);
  }
  function reset() {
    setFrom(defaults.from); setTo(defaults.to); setSupplierID(''); setError('');
    router.replace('/dashboard');
  }

  const metrics = [
    ['Pending Approval', data.metrics.pendingApproval, 'Waiting for director'],
    ['Open PO', data.metrics.openPO, 'Approved or partially received'],
    ['Received Kanban', data.metrics.receivedKanban, 'Within selected period'],
    ['Outstanding Kanban', data.metrics.outstandingKanban, 'Not yet received'],
    ['Current Stock', data.metrics.currentStock, 'Kanban currently in stock'],
  ] as const;

  return <section className="dashboard-page">
    <div className="page-title-row"><div><h1>Dashboard</h1><p className="muted">Procurement and warehouse operational summary.</p></div><span className="live-badge"><i /> Live data</span></div>
    <form className="dashboard-filters" onSubmit={apply}>
      <label><span>From Date</span><input aria-label="From Date" type="date" value={from} onChange={event => setFrom(event.target.value)} /></label>
      <label><span>To Date</span><input aria-label="To Date" type="date" value={to} onChange={event => setTo(event.target.value)} /></label>
      <label><span>Supplier</span><select aria-label="Supplier" value={supplierID} onChange={event => setSupplierID(event.target.value)}><option value="">All suppliers</option>{suppliers.map(item => <option key={item.id} value={item.id}>{item.code} — {item.name}</option>)}</select></label>
      <button className="primary-button">Apply</button><button type="button" onClick={reset}>Reset</button>
    </form>
    {error ? <div className="dashboard-alert" role="alert"><span>{error}</span>{error === 'Dashboard could not be loaded.' ? <button onClick={load}>Retry</button> : null}</div> : null}
    <div className={`metric-grid metric-grid--five${loading ? ' dashboard-loading' : ''}`}>
      {metrics.map(([label, value, note]) => <article key={label}><span>{label}</span><strong>{value}</strong><small>{note}</small></article>)}
    </div>
    <div className="dashboard-grid">
      <DashboardChart title="PO & Receiving Trend" subtitle={`${from} to ${to}`} empty={!loading && data.trend.every(point => point.ordered === 0 && point.received === 0)}>{loading ? <div className="chart-loading" /> : <TrendChart data={data.trend} />}</DashboardChart>
      <DashboardChart title="PO Status" subtitle="Current distribution" empty={!loading && data.poStatus.length === 0} variant="donut">{loading ? <div className="chart-loading" /> : <StatusDonut data={data.poStatus} />}</DashboardChart>
      <DashboardChart title="Outstanding by Supplier" subtitle="Top 10 suppliers" empty={!loading && data.outstandingBySupplier.length === 0} variant="bars">{loading ? <div className="chart-loading" /> : <SupplierBars data={data.outstandingBySupplier} />}</DashboardChart>
      <article className="dashboard-panel activity-panel"><div className="panel-heading"><div><h2>Latest Activity</h2><p>Recent operational updates</p></div></div>
        <div className="activity-list">{loading ? <div className="chart-loading" /> : data.activities.length ? data.activities.map(item => <div key={`${item.type}-${item.id}`}><i>{item.type.slice(0, 1)}</i><span><strong>{item.label}</strong><small>{new Date(item.occurredAt).toLocaleString()}</small></span></div>) : <div className="activity-empty"><strong>No activity in this period</strong><small>Try a wider date range.</small></div>}</div>
      </article>
    </div>
  </section>;
}
