'use client';

import Link from 'next/link';
import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useCurrentUser } from '../app-shell/app-shell';

export type PurchaseOrderApproval = {
  id: string;
  purchaseOrderId: string;
  version: number;
  status: string;
  createdAt: string;
  createdBy?: { displayName?: string };
  poNumber?: string;
  supplierName?: string;
};

type ApprovalResponse = { items?: PurchaseOrderApproval[]; total?: number };
type PurchaseOrder = { id: string; poNumber: string; supplierId: string };
type Supplier = { id: string; code: string; name: string };
type Decision = { approval: PurchaseOrderApproval; action: 'approve' | 'reject' };
const refreshEvent = 'purchase-order-approvals:refresh';

function approvalReference(approval: PurchaseOrderApproval) { return approval.poNumber ?? `PO ${approval.purchaseOrderId}`; }

async function responseMessage(response: Response) {
  try {
    const body = await response.json() as { message?: string; fields?: Record<string, string> };
    return body.message ?? body.fields?._form ?? Object.values(body.fields ?? {})[0] ?? 'Approval could not be updated';
  } catch { return 'Approval could not be updated'; }
}

async function enrichApprovals(approvals: PurchaseOrderApproval[], permissions: string[]) {
  if (!permissions.includes('po.view') || approvals.length === 0) return approvals;
  const orders = await Promise.all(approvals.map(async approval => {
    try {
      const response = await fetch(`/api/purchase-orders/${approval.purchaseOrderId}`, { credentials: 'include' });
      return response.ok ? await response.json() as PurchaseOrder : null;
    } catch { return null; }
  }));
  const byID = new Map(orders.filter((order): order is PurchaseOrder => Boolean(order)).map(order => [order.id, order]));
  let suppliers = new Map<string, Supplier>();
  if (permissions.includes('master_data.view')) {
    try {
      const response = await fetch('/api/master-data/suppliers?limit=200', { credentials: 'include' });
      if (response.ok) suppliers = new Map(((await response.json() as { items?: Supplier[] }).items ?? []).map(supplier => [supplier.id, supplier]));
    } catch { /* PO details remain useful without supplier directory access. */ }
  }
  return approvals.map(approval => {
    const order = byID.get(approval.purchaseOrderId); const supplier = order ? suppliers.get(order.supplierId) : undefined;
    return { ...approval, poNumber: order?.poNumber, supplierName: supplier ? `${supplier.code} — ${supplier.name}` : order?.supplierId };
  });
}

export function ApprovalInbox() {
  const user = useCurrentUser();
  const [items, setItems] = useState<PurchaseOrderApproval[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [savingID, setSavingID] = useState('');
  const [rejecting, setRejecting] = useState<Decision | null>(null);
  const [reason, setReason] = useState('');
  const [reasonError, setReasonError] = useState('');
  const canApprove = Boolean(user?.permissions.includes('po.approve'));
  const canReject = Boolean(user?.permissions.includes('po.reject'));
  const canView = Boolean(user?.permissions.includes('po.view'));

  const load = useCallback(async () => {
    if (!canApprove) { setItems([]); setLoading(false); return; }
    setLoading(true); setError('');
    try {
      const response = await fetch('/api/purchase-order-approvals', { credentials: 'include' });
      if (!response.ok) throw new Error(await responseMessage(response));
      const payload = await response.json() as ApprovalResponse;
      setItems(await enrichApprovals(payload.items ?? [], user?.permissions ?? []));
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Approvals could not be loaded'); }
    finally { setLoading(false); }
  }, [canApprove]);

  useEffect(() => {
    void load();
    const refresh = () => void load();
    window.addEventListener(refreshEvent, refresh);
    return () => window.removeEventListener(refreshEvent, refresh);
  }, [load]);

  async function decide(approval: PurchaseOrderApproval, action: 'approve' | 'reject', decisionReason = '') {
    setSavingID(approval.id); setError('');
    try {
      const response = await fetch(`/api/purchase-order-approvals/${approval.id}/${action}`, {
        method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(action === 'reject' ? { reason: decisionReason } : {})
      });
      if (!response.ok) throw new Error(await responseMessage(response));
      setItems(value => value.filter(item => item.id !== approval.id));
      window.dispatchEvent(new Event(refreshEvent));
      setRejecting(null); setReason(''); setReasonError('');
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Approval could not be updated'); }
    finally { setSavingID(''); }
  }

  function submitDecision(event: FormEvent) {
    event.preventDefault();
    if (!rejecting) return;
    if (rejecting.action === 'reject' && !reason.trim()) { setReasonError('Rejection reason is required'); return; }
    void decide(rejecting.approval, rejecting.action, reason.trim());
  }

  if (!canApprove) return <section className="module-index approval-inbox"><div className="table-empty" role="status"><strong>Approval permission is required</strong><span>You do not have access to pending purchase order approvals.</span></div></section>;

  return <section className="module-index approval-inbox">
    <div className="page-title-row"><div><h1>Approval Inbox</h1><p className="muted">Review pending supplier orders.</p></div></div>
    {error ? <div className="table-empty" role="alert"><strong>Could not update approval</strong><span>{error}</span><button className="table-action" onClick={() => void load()}>Retry</button></div> : null}
    <div className="table-frame"><table><thead><tr><th>Purchase Order</th><th>Requested By</th><th>Requested</th><th>Actions</th></tr></thead><tbody>
      {loading ? <tr><td colSpan={4}><div className="table-empty" role="status">Loading...</div></td></tr> : items.length === 0 ? <tr><td colSpan={4}><div className="table-empty" role="status"><strong>No purchase orders awaiting approval</strong><span>New approval requests will appear here.</span></div></td></tr> : items.map(approval => <tr key={approval.id}>
        <td>{canView ? <Link aria-label={`Open ${approvalReference(approval)}`} href={`/supplier-orders/${approval.purchaseOrderId}`}>{approvalReference(approval)}</Link> : approvalReference(approval)}<small>{approval.supplierName ?? 'Supplier unavailable'}</small></td>
        <td>{approval.createdBy?.displayName ?? '—'}</td><td>{approval.createdAt.slice(0, 10) || '—'}</td>
        <td><div className="approval-actions">{canApprove ? <button className="primary-button" disabled={savingID === approval.id} aria-label={`Approve ${approvalReference(approval)}`} onClick={() => { setRejecting({ approval, action: 'approve' }); setReason(''); setReasonError(''); }}>Approve</button> : null}{canReject ? <button disabled={savingID === approval.id} aria-label={`Reject ${approvalReference(approval)}`} onClick={() => { setRejecting({ approval, action: 'reject' }); setReason(''); setReasonError(''); }}>Reject</button> : null}</div></td>
      </tr>)}
    </tbody></table></div>
    {rejecting ? <><button className="crud-scrim" aria-label="Close decision form" onClick={() => !savingID && setRejecting(null)} /><div className="crud-modal crud-modal--compact" role="dialog" aria-modal="true" aria-label={`${rejecting.action === 'approve' ? 'Approve' : 'Reject'} ${approvalReference(rejecting.approval)}`}><div className="crud-modal-heading"><div><strong>{rejecting.action === 'approve' ? 'Approve' : 'Reject'} {approvalReference(rejecting.approval)}</strong><span>{rejecting.action === 'approve' ? 'This decision cannot be undone.' : 'Provide a brief reason for the requester.'}</span></div><button aria-label="Close decision form" onClick={() => !savingID && setRejecting(null)}>×</button></div><form onSubmit={submitDecision}>{error ? <p className="form-error" role="alert">{error}</p> : null}{rejecting.action === 'reject' ? <div className="crud-fields"><label className="approval-reason"><span>Rejection reason *</span><textarea aria-label="Rejection reason" value={reason} onChange={event => { setReason(event.target.value); setReasonError(''); }} autoFocus />{reasonError ? <small className="form-error">{reasonError}</small> : null}</label></div> : <div className="crud-fields"><p>Confirm approval of this purchase order.</p></div>}<div className="crud-actions"><button type="button" onClick={() => setRejecting(null)} disabled={Boolean(savingID)}>Cancel</button><button className="primary-button" disabled={Boolean(savingID)}>{savingID ? `${rejecting.action === 'approve' ? 'Approving' : 'Rejecting'}...` : `${rejecting.action === 'approve' ? 'Approve' : 'Reject'} order`}</button></div></form></div></> : null}
  </section>;
}
