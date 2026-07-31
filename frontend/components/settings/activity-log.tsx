'use client';

import { useEffect, useMemo, useState } from 'react';
import { TablePagination } from '../table-pagination';

type Snapshot = Record<string, unknown>;
type ActivityItem = {
  id: string;
  occurredAt: string;
  actorUserId?: string;
  actorName: string;
  module: string;
  action: string;
  targetType: string;
  targetId?: string;
  targetCode: string;
  before?: Snapshot;
  after?: Snapshot;
};
type ActorOption = { id: string; name: string };
type FilterOptions = { actors: ActorOption[]; modules: string[]; actions: string[] };
type ActivityPage = {
  items: ActivityItem[];
  total: number;
  page: number;
  pageSize: number;
  filters: FilterOptions;
};

const emptyFilters: FilterOptions = { actors: [], modules: [], actions: [] };

function dateInput(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function initialDates() {
  const to = new Date();
  const from = new Date(to);
  from.setDate(from.getDate() - 29);
  return { from: dateInput(from), to: dateInput(to) };
}

function readable(value: unknown) {
  if (value === null || value === undefined || value === '') return '—';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

function label(value: string) {
  return value.toLowerCase().replaceAll('_', ' ').replace(/\b\w/g, letter => letter.toUpperCase());
}

function actionClass(action: string) {
  if (['APPROVED', 'COMPLETED', 'RECEIVED', 'ACTIVATED'].includes(action)) return 'status-pill--green';
  if (['REJECTED', 'CANCELLED', 'DEACTIVATED'].includes(action)) return 'status-pill--red';
  if (['SUBMITTED', 'ISSUED'].includes(action)) return 'status-pill--amber';
  return 'status-pill--blue';
}

function changes(item: ActivityItem) {
  const before = item.before ?? {};
  const after = item.after ?? {};
  return [...new Set([...Object.keys(before), ...Object.keys(after)])]
    .sort()
    .filter(key => JSON.stringify(before[key]) !== JSON.stringify(after[key]))
    .map(key => ({ key, before: before[key], after: after[key] }));
}

export function ActivityLog() {
  const dates = useMemo(initialDates, []);
  const [from, setFrom] = useState(dates.from);
  const [to, setTo] = useState(dates.to);
  const [userId, setUserId] = useState('');
  const [module, setModule] = useState('');
  const [action, setAction] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [items, setItems] = useState<ActivityItem[]>([]);
  const [total, setTotal] = useState(0);
  const [filters, setFilters] = useState<FilterOptions>(emptyFilters);
  const [selected, setSelected] = useState<ActivityItem | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const controller = new AbortController();
    const params = new URLSearchParams({ from, to, page: String(page), pageSize: String(pageSize) });
    if (userId) params.set('userId', userId);
    if (module) params.set('module', module);
    if (action) params.set('action', action);
    setLoading(true);
    setError('');
    fetch(`/api/activity-logs?${params}`, { credentials: 'include', signal: controller.signal })
      .then(async response => {
        if (!response.ok) throw new Error('load failed');
        return response.json() as Promise<ActivityPage>;
      })
      .then(payload => {
        setItems(Array.isArray(payload.items) ? payload.items : []);
        setTotal(Number(payload.total) || 0);
        setFilters({
          actors: payload.filters?.actors ?? [],
          modules: payload.filters?.modules ?? [],
          actions: payload.filters?.actions ?? []
        });
      })
      .catch(requestError => {
        if ((requestError as Error).name !== 'AbortError') {
          setItems([]);
          setTotal(0);
          setError('Activity log could not be loaded.');
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [from, to, userId, module, action, page, pageSize]);

  const resetPage = (setter: (value: string) => void) => (value: string) => {
    setter(value);
    setPage(1);
  };
  const detailChanges = selected ? changes(selected) : [];

  return <section className="module-index activity-log">
    <div className="page-title-row">
      <div>
        <h1>Activity Log</h1>
        <p className="muted">Immutable history of important data and workflow changes.</p>
      </div>
      <span className="activity-log-readonly">Read only</span>
    </div>

    <div className="activity-filter-bar">
      <label>From<input aria-label="From date" type="date" value={from} max={to} onChange={event => resetPage(setFrom)(event.target.value)} /></label>
      <label>To<input aria-label="To date" type="date" value={to} min={from} onChange={event => resetPage(setTo)(event.target.value)} /></label>
      <label>User<select aria-label="User" value={userId} onChange={event => resetPage(setUserId)(event.target.value)}>
        <option value="">All users</option>
        {filters.actors.map(actor => <option key={actor.id} value={actor.id}>{actor.name}</option>)}
      </select></label>
      <label>Module<select aria-label="Module" value={module} onChange={event => resetPage(setModule)(event.target.value)}>
        <option value="">All modules</option>
        {filters.modules.map(value => <option key={value} value={value}>{label(value)}</option>)}
      </select></label>
      <label>Action<select aria-label="Action" value={action} onChange={event => resetPage(setAction)(event.target.value)}>
        <option value="">All actions</option>
        {filters.actions.map(value => <option key={value} value={value}>{label(value)}</option>)}
      </select></label>
      <span className="activity-record-count">{total} records</span>
    </div>

    <div className="table-frame">
      <table>
        <thead><tr><th>Date &amp; Time</th><th>User</th><th>Module</th><th>Activity</th><th>Target</th><th aria-label="Details" /></tr></thead>
        <tbody>
          {error ? <tr><td colSpan={6}><div className="table-empty form-error" role="alert">{error}</div></td></tr>
            : loading ? <tr><td colSpan={6}><div className="table-empty">Loading activity...</div></td></tr>
              : items.length === 0 ? <tr><td colSpan={6}><div className="table-empty">No activity matches these filters.</div></td></tr>
                : items.map(item => <tr key={item.id}>
                  <td>{new Date(item.occurredAt).toLocaleString('en-GB', { dateStyle: 'medium', timeStyle: 'short' })}</td>
                  <td><strong className="activity-actor">{item.actorName}</strong></td>
                  <td>{label(item.module)}</td>
                  <td><span className={`status-pill ${actionClass(item.action)}`}>{label(item.action)}</span></td>
                  <td><span className="activity-target">{item.targetCode || item.targetType}</span><small>{label(item.targetType)}</small></td>
                  <td><button className="table-action" aria-label={`View details for ${item.targetCode || item.targetType}`} onClick={() => setSelected(item)}>Details</button></td>
                </tr>)}
        </tbody>
      </table>
      <TablePagination page={page} pageSize={pageSize} total={total} onPageChange={setPage} onPageSizeChange={size => { setPageSize(size); setPage(1); }} />
    </div>

    {selected ? <>
      <button className="crud-scrim" aria-label="Close activity details" onClick={() => setSelected(null)} />
      <div className="crud-modal crud-modal--wide activity-detail" role="dialog" aria-modal="true" aria-label="Activity details" onKeyDown={event => {
        if (event.key === 'Escape') setSelected(null);
      }}>
        <div className="crud-modal-heading">
          <div><strong>{label(selected.action)} · {selected.targetCode || label(selected.targetType)}</strong><span>{selected.actorName} · {new Date(selected.occurredAt).toLocaleString('en-GB')}</span></div>
          <button aria-label="Close activity details" onClick={() => setSelected(null)}>×</button>
        </div>
        <div className="activity-detail-body">
          <dl className="activity-detail-summary">
            <div><dt>Module</dt><dd>{label(selected.module)}</dd></div>
            <div><dt>Target Type</dt><dd>{label(selected.targetType)}</dd></div>
            <div><dt>Target ID</dt><dd>{selected.targetId ?? '—'}</dd></div>
          </dl>
          <div className="activity-change-table table-detail">
            <table>
              <thead><tr><th>Field</th><th>Before</th><th>After</th></tr></thead>
              <tbody>{detailChanges.length === 0
                ? <tr><td colSpan={3}>No field-level difference is available.</td></tr>
                : detailChanges.map(change => <tr key={change.key}><th>{label(change.key)}</th><td>{readable(change.before)}</td><td>{readable(change.after)}</td></tr>)}</tbody>
            </table>
          </div>
        </div>
      </div>
    </> : null}
  </section>;
}
