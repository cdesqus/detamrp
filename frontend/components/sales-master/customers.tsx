'use client';

import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useCurrentUser } from '../app-shell/app-shell';

type Customer = {
  id: string;
  code: string;
  name: string;
  address?: string;
  contact?: string;
  email?: string;
  phone?: string;
  active: boolean;
};

type CustomerForm = Omit<Customer, 'id'>;
type Props = { permissions?: string[] };

const emptyCustomer: CustomerForm = {
  code: '', name: '', address: '', contact: '', email: '', phone: '', active: true
};

export function Customers({ permissions }: Props = {}) {
  const currentUser = useCurrentUser();
  const permissionList = permissions ?? currentUser?.permissions ?? [];
  const canManage = permissionList.includes('customer.manage');
  const [items, setItems] = useState<Customer[]>([]);
  const [total, setTotal] = useState(0);
  const [ready, setReady] = useState(false);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [active, setActive] = useState('true');
  const [editing, setEditing] = useState<Customer | null>(null);
  const [form, setForm] = useState<CustomerForm>(emptyCustomer);
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const load = useCallback(async () => {
    setError('');
    try {
      const query = new URLSearchParams({ search });
      if (active !== 'all') query.set('active', active);
      const response = await fetch(`/api/customers?${query}`, { credentials: 'include' });
      if (!response.ok) throw new Error('Customers could not be loaded');
      const body = await response.json() as { items: Customer[]; total: number };
      setItems(body.items);
      setTotal(body.total);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Customers could not be loaded');
    } finally {
      setReady(true);
    }
  }, [active, search]);

  useEffect(() => {
    const timer = setTimeout(load, 250);
    return () => clearTimeout(timer);
  }, [load]);

  function begin(customer?: Customer) {
    setEditing(customer ?? null);
    setForm(customer ? {
      code: customer.code,
      name: customer.name,
      address: customer.address ?? '',
      contact: customer.contact ?? '',
      email: customer.email ?? '',
      phone: customer.phone ?? '',
      active: customer.active
    } : emptyCustomer);
    setFieldErrors({});
    setOpen(true);
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setFieldErrors({});
    try {
      const response = await fetch(`/api/customers${editing ? `/${editing.id}` : ''}`, {
        method: editing ? 'PATCH' : 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form)
      });
      if (!response.ok) {
        const body = await response.json() as { message?: string; fields?: Record<string, string> };
        setFieldErrors(body.fields ?? { _form: body.message ?? 'Customer could not be saved' });
        return;
      }
      setOpen(false);
      await load();
    } finally {
      setSaving(false);
    }
  }

  if (!ready) return <section className="module-index"><p className="muted">Loading customers...</p></section>;

  return <section className="module-index">
    <div className="page-title-row"><div><h1>Customers</h1><p className="muted">Customer identities and delivery contact details.</p></div>{canManage && <button className="primary-button" onClick={() => begin()}>New customer</button>}</div>
    <div className="table-toolbar"><input type="search" value={search} onChange={event => setSearch(event.target.value)} aria-label="Search customers" placeholder="Search code, name, or contact"/><div className="toolbar-actions"><select aria-label="Status filter" value={active} onChange={event => setActive(event.target.value)}><option value="true">Active</option><option value="false">Inactive</option><option value="all">All status</option></select><span>{total} records</span></div></div>
    {error ? <div className="table-empty"><strong>Could not load data</strong><span>{error}</span></div> : <div className="table-frame"><table><thead><tr><th>Customer Code</th><th>Customer Name</th><th>Contact</th><th>Email</th><th>Phone</th><th>Status</th>{canManage && <th>Action</th>}</tr></thead><tbody>{items.length === 0 ? <tr><td className="table-row-empty" colSpan={canManage ? 7 : 6}><div className="table-empty"><strong>No customers yet</strong></div></td></tr> : items.map(customer => <tr key={customer.id}><td>{customer.code}</td><td>{customer.name}</td><td>{customer.contact || '-'}</td><td>{customer.email || '-'}</td><td>{customer.phone || '-'}</td><td>{customer.active ? 'Active' : 'Inactive'}</td>{canManage && <td><button className="table-action" onClick={() => begin(customer)}>Edit</button></td>}</tr>)}</tbody></table></div>}
    {open && <><button className="crud-scrim" aria-label="Close form" onClick={() => setOpen(false)}/><div className="crud-modal crud-modal--wide" role="dialog" aria-modal="true" aria-label={`${editing ? 'Edit' : 'New'} customer`}><div className="crud-modal-heading"><div><strong>{editing ? 'Edit' : 'New'} customer</strong><span>Enter customer identity and contact information.</span></div><button aria-label="Close form" onClick={() => setOpen(false)}>×</button></div><form onSubmit={submit}>{fieldErrors._form && <p className="form-error">{fieldErrors._form}</p>}<div className="crud-fields">
      <TextField label="Customer Code" value={form.code} error={fieldErrors.code} required onChange={value => setForm(current => ({ ...current, code: value }))}/>
      <TextField label="Customer Name" value={form.name} error={fieldErrors.name} required onChange={value => setForm(current => ({ ...current, name: value }))}/>
      <TextField label="Contact Person" value={form.contact ?? ''} onChange={value => setForm(current => ({ ...current, contact: value }))}/>
      <TextField label="Email" type="email" value={form.email ?? ''} error={fieldErrors.email} onChange={value => setForm(current => ({ ...current, email: value }))}/>
      <TextField label="Phone" value={form.phone ?? ''} onChange={value => setForm(current => ({ ...current, phone: value }))}/>
      <label><span>Address</span><textarea value={form.address} onChange={event => setForm(current => ({ ...current, address: event.target.value }))}/></label>
      <label className="check-field"><input type="checkbox" checked={form.active} onChange={event => setForm(current => ({ ...current, active: event.target.checked }))}/> Active</label>
    </div><div className="crud-actions"><button type="button" onClick={() => setOpen(false)}>Cancel</button><button className="primary-button" disabled={saving}>{saving ? 'Saving...' : 'Save'}</button></div></form></div></>}
  </section>;
}

function TextField({ label, value, onChange, type = 'text', required = false, error }: { label: string; value: string; onChange: (value: string) => void; type?: string; required?: boolean; error?: string }) {
  return <label><span>{label}{required ? ' *' : ''}</span><input type={type} value={value} required={required} onChange={event => onChange(event.target.value)}/>{error && <small>{error}</small>}</label>;
}
