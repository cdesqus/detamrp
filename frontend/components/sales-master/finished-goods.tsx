'use client';

import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useCurrentUser } from '../app-shell/app-shell';

type FinishedGood = {
  id: string;
  itemCode: string;
  name: string;
  baseUnitId: string;
  baseUnitCode: string;
  baseUnitName: string;
  salesPrice: string;
  currency: string;
  priceVersion: number;
  active: boolean;
};
type Unit = { id: string; code: string; name: string; active: boolean };
type FinishedGoodForm = Pick<FinishedGood, 'itemCode' | 'name' | 'baseUnitId' | 'salesPrice' | 'currency' | 'active'>;
type Props = { permissions?: string[] };

const emptyFinishedGood: FinishedGoodForm = {
  itemCode: '', name: '', baseUnitId: '', salesPrice: '', currency: 'IDR', active: true
};
const currencies = ['IDR', 'USD', 'EUR', 'JPY', 'SGD'];

export function FinishedGoods({ permissions }: Props = {}) {
  const currentUser = useCurrentUser();
  const permissionList = permissions ?? currentUser?.permissions ?? [];
  const canManage = permissionList.includes('fg.manage');
  const canManagePrice = permissionList.includes('fg.price.manage');
  const [items, setItems] = useState<FinishedGood[]>([]);
  const [units, setUnits] = useState<Unit[]>([]);
  const [total, setTotal] = useState(0);
  const [ready, setReady] = useState(false);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [active, setActive] = useState('true');
  const [editing, setEditing] = useState<FinishedGood | null>(null);
  const [form, setForm] = useState<FinishedGoodForm>(emptyFinishedGood);
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const load = useCallback(async () => {
    setError('');
    try {
      const query = new URLSearchParams({ search });
      if (active !== 'all') query.set('active', active);
      const [goodsResponse, unitsResponse] = await Promise.all([
        fetch(`/api/finished-goods?${query}`, { credentials: 'include' }),
        fetch('/api/master-data/units?active=true&limit=200', { credentials: 'include' })
      ]);
      if (!goodsResponse.ok || !unitsResponse.ok) throw new Error('Finished goods could not be loaded');
      const goodsBody = await goodsResponse.json() as { items: FinishedGood[]; total: number };
      const unitsBody = await unitsResponse.json() as { items: Unit[] };
      setItems(goodsBody.items);
      setTotal(goodsBody.total);
      setUnits(unitsBody.items);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Finished goods could not be loaded');
    } finally {
      setReady(true);
    }
  }, [active, search]);

  useEffect(() => {
    const timer = setTimeout(load, 250);
    return () => clearTimeout(timer);
  }, [load]);

  function begin(good?: FinishedGood) {
    setEditing(good ?? null);
    setForm(good ? {
      itemCode: good.itemCode,
      name: good.name,
      baseUnitId: good.baseUnitId,
      salesPrice: good.salesPrice,
      currency: good.currency,
      active: good.active
    } : emptyFinishedGood);
    setFieldErrors({});
    setOpen(true);
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setFieldErrors({});
    try {
      const response = await fetch(`/api/finished-goods${editing ? `/${editing.id}` : ''}`, {
        method: editing ? 'PATCH' : 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form)
      });
      if (!response.ok) {
        const body = await response.json() as { message?: string; fields?: Record<string, string> };
        setFieldErrors(body.fields ?? { _form: body.message ?? 'Finished good could not be saved' });
        return;
      }
      setOpen(false);
      await load();
    } finally {
      setSaving(false);
    }
  }

  if (!ready) return <section className="module-index"><p className="muted">Loading finished goods...</p></section>;

  return <section className="module-index">
    <div className="page-title-row"><div><h1>Finished Goods</h1><p className="muted">Finished items, base units, and the current master sales price.</p></div>{canManage && canManagePrice && <button className="primary-button" onClick={() => begin()}>New finished good</button>}</div>
    <div className="table-toolbar"><input type="search" value={search} onChange={event => setSearch(event.target.value)} aria-label="Search finished goods" placeholder="Search Item Code or finished good"/><div className="toolbar-actions"><select aria-label="Status filter" value={active} onChange={event => setActive(event.target.value)}><option value="true">Active</option><option value="false">Inactive</option><option value="all">All status</option></select><span>{total} records</span></div></div>
    {error ? <div className="table-empty"><strong>Could not load data</strong><span>{error}</span></div> : <div className="table-frame"><table><thead><tr><th>Item Code</th><th>Finished Good</th><th>Base Unit</th><th>Master Price</th><th>Currency</th><th>Price Version</th><th>Status</th>{canManage && <th>Action</th>}</tr></thead><tbody>{items.length === 0 ? <tr><td className="table-row-empty" colSpan={canManage ? 8 : 7}><div className="table-empty"><strong>No finished goods yet</strong></div></td></tr> : items.map(good => <tr key={good.id}><td>{good.itemCode}</td><td>{good.name}</td><td>{good.baseUnitCode}</td><td>{good.salesPrice}</td><td>{good.currency}</td><td>{good.priceVersion}</td><td>{good.active ? 'Active' : 'Inactive'}</td>{canManage && <td><button className="table-action" onClick={() => begin(good)}>Edit</button></td>}</tr>)}</tbody></table></div>}
    {open && <><button className="crud-scrim" aria-label="Close form" onClick={() => setOpen(false)}/><div className="crud-modal crud-modal--wide" role="dialog" aria-modal="true" aria-label={`${editing ? 'Edit' : 'New'} finished good`}><div className="crud-modal-heading"><div><strong>{editing ? 'Edit' : 'New'} finished good</strong><span>Define the reusable finished item and its master price.</span></div><button aria-label="Close form" onClick={() => setOpen(false)}>×</button></div><form onSubmit={submit}>{fieldErrors._form && <p className="form-error">{fieldErrors._form}</p>}<div className="crud-fields">
      <label><span>Item Code *</span><input value={form.itemCode} required onChange={event => setForm(current => ({ ...current, itemCode: event.target.value }))}/>{fieldErrors.itemCode && <small>{fieldErrors.itemCode}</small>}</label>
      <label><span>Finished Good Name *</span><input value={form.name} required onChange={event => setForm(current => ({ ...current, name: event.target.value }))}/>{fieldErrors.name && <small>{fieldErrors.name}</small>}</label>
      <label><span>Base Unit *</span><select value={form.baseUnitId} required onChange={event => setForm(current => ({ ...current, baseUnitId: event.target.value }))}><option value="">Select...</option>{units.map(unit => <option key={unit.id} value={unit.id}>{unit.code} — {unit.name}</option>)}</select>{fieldErrors.baseUnitId && <small>{fieldErrors.baseUnitId}</small>}</label>
      <label><span>Sales Price *</span><input type="number" min="0.000001" step="0.000001" value={form.salesPrice} required disabled={!canManagePrice} onChange={event => setForm(current => ({ ...current, salesPrice: event.target.value }))}/>{fieldErrors.salesPrice && <small>{fieldErrors.salesPrice}</small>}</label>
      <label><span>Currency *</span><select value={form.currency} disabled={!canManagePrice} required onChange={event => setForm(current => ({ ...current, currency: event.target.value }))}>{currencies.map(currency => <option key={currency} value={currency}>{currency}</option>)}</select>{fieldErrors.currency && <small>{fieldErrors.currency}</small>}</label>
      {!canManagePrice && <p className="muted">Price changes require the FG Price Manage permission.</p>}
      <label className="check-field"><input type="checkbox" checked={form.active} onChange={event => setForm(current => ({ ...current, active: event.target.checked }))}/> Active</label>
    </div><div className="crud-actions"><button type="button" onClick={() => setOpen(false)}>Cancel</button><button className="primary-button" disabled={saving}>{saving ? 'Saving...' : 'Save'}</button></div></form></div></>}
  </section>;
}
