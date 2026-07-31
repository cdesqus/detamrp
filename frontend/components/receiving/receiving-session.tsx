'use client';

import { FormEvent, useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';

type Scan = { kanbanLotId: string; kanbanId: string; materialCode: string; materialName: string; unit: string; quantity: string };
type Session = { id: string; receivingNumber: string; deliveryNoteNumber: string; poNumber: string; supplierName: string; status: string; planned: number; previouslyReceived: number; outstanding: number; scans: Scan[] };

const scanMessages: Record<string, string> = {
  KANBAN_ALREADY_SCANNED: 'Kanban has already been scanned in this session.',
  KANBAN_ALREADY_RECEIVED: 'Kanban has already been received.',
  KANBAN_WRONG_DN: 'Kanban does not belong to this Delivery Note.',
  KANBAN_NOT_FOUND: 'Kanban ID was not found.',
};

export function ReceivingSession({ id }: { id: string }) {
  const router = useRouter();
  const [data, setData] = useState<Session | null>(null);
  const [value, setValue] = useState('');
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [busy, setBusy] = useState(false);
  const input = useRef<HTMLInputElement>(null);

  const load = () => fetch(`/api/receiving-sessions/${id}`, { credentials: 'include' })
    .then(response => { if (!response.ok) throw new Error(); return response.json(); })
    .then(payload => setData({ ...payload, scans: Array.isArray(payload.scans) ? payload.scans : [] }))
    .catch(() => setError('Receiving session could not be loaded.'));

  useEffect(() => { void load(); }, [id]);
  useEffect(() => { input.current?.focus(); }, [data?.scans?.length]);

  async function scan(event: FormEvent) {
    event.preventDefault();
    if (!value.trim() || busy) return;
    setBusy(true); setError(''); setMessage('');
    try {
      const response = await fetch(`/api/receiving-sessions/${id}/scans`, {
        method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ kanbanId: value }),
      });
      const payload = await response.json().catch(() => ({})) as Session & { code?: string; error?: string };
      if (!response.ok) throw new Error(scanMessages[payload.code ?? ''] ?? payload.error ?? 'Kanban could not be scanned.');
      setData({ ...payload, scans: Array.isArray(payload.scans) ? payload.scans : [] });
      setValue(''); setMessage('Kanban scanned successfully.');
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Kanban could not be scanned.');
    } finally {
      setBusy(false);
      setTimeout(() => input.current?.focus(), 0);
    }
  }

  async function action(name: string) {
    setBusy(true); setError(''); setMessage('');
    const response = await fetch(`/api/receiving-sessions/${id}/${name}`, { method: 'POST', credentials: 'include' });
    setBusy(false);
    if (!response.ok) { setError('Action failed.'); return; }
    if (name === 'complete') router.push('/receiving');
    else setData(await response.json());
  }

  if (!data) return <div className="table-empty">{error || 'Loading receiving session...'}</div>;
  return <section className="scan-session">
    <div className="scan-session-header"><button className="table-action" onClick={() => router.push('/receiving')}>Back</button><div><h1>{data.receivingNumber}</h1><p>{data.deliveryNoteNumber} · {data.poNumber} · {data.supplierName}</p></div><span className="status-pill status-pill--blue">{data.status}</span></div>
    <div className="scan-focus-zone"><span>Scan or Type Kanban ID</span><form onSubmit={scan}><input ref={input} aria-label="Scan or Type Kanban ID" value={value} onChange={event => setValue(event.target.value)} disabled={data.status !== 'ACTIVE' || busy} autoComplete="off" /><button className="primary-button" disabled={!value.trim() || busy}>Add</button></form>{error && <p className="form-error" role="alert">{error}</p>}{message && <p className="scan-success" role="status">{message}</p>}<div className="scan-counters"><span>Scanned <b>{data.scans?.length ?? 0}</b></span><span>Outstanding before session <b>{data.outstanding}</b></span></div></div>
    <div className="table-frame table-detail"><table><thead><tr><th>Kanban ID</th><th>Raw Material</th><th>Quantity</th><th>Unit</th><th className="table-column-actions"></th></tr></thead><tbody>{(data.scans ?? []).length === 0 ? <tr><td className="table-row-empty" colSpan={5}><div className="table-empty">Ready to scan.</div></td></tr> : (data.scans ?? []).map(item => <tr key={item.kanbanLotId}><td>{item.kanbanId}</td><td>{item.materialCode} — {item.materialName}</td><td>{item.quantity}</td><td>{item.unit}</td><td><button className="table-action" onClick={async () => { await fetch(`/api/receiving-sessions/${id}/scans/${item.kanbanLotId}`, { method: 'DELETE', credentials: 'include' }); setMessage(''); void load(); }}>Remove</button></td></tr>)}</tbody></table></div>
    <div className="supplier-order-actions"><button disabled={busy} onClick={() => action('pause')}>Pause session</button><button className="primary-button" disabled={busy || (data.scans ?? []).length === 0} onClick={() => action('complete')}>Complete receiving</button></div>
  </section>;
}
