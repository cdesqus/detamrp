'use client';

import Link from 'next/link';
import { FormEvent, MouseEvent, useCallback, useEffect, useRef, useState } from 'react';
import { useCurrentUser } from '../app-shell/app-shell';

export type PurchaseOrderApproval = {
  id: string;
  purchaseOrderId: string;
  poNumber?: string;
  supplierId?: string;
  supplierName?: string;
  version: number;
  status: string;
  createdAt: string;
  createdBy?: { displayName?: string };
};

type ApprovalResponse = { items?: PurchaseOrderApproval[]; total?: number };
type Decision = { approval: PurchaseOrderApproval; action: 'approve' | 'reject' };
const refreshEvent = 'purchase-order-approvals:refresh';
const reasonErrorID = 'approval-rejection-reason-error';
const pageLimit = 50;

function approvalReference(approval: PurchaseOrderApproval) { return approval.poNumber ?? `PO ${approval.purchaseOrderId}`; }

async function responseMessage(response: Response) {
  try {
    const body = await response.json() as { message?: string; fields?: Record<string, string> };
    return body.message ?? body.fields?._form ?? Object.values(body.fields ?? {})[0] ?? 'Approval could not be updated';
  } catch { return 'Approval could not be updated'; }
}

export function ApprovalInbox() {
  const user = useCurrentUser();
  const [items, setItems] = useState<PurchaseOrderApproval[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(true);
  const [hasSnapshot, setHasSnapshot] = useState(false);
  const [loadError, setLoadError] = useState('');
  const [decisionError, setDecisionError] = useState('');
  const [savingID, setSavingID] = useState('');
  const [decision, setDecision] = useState<Decision | null>(null);
  const [reason, setReason] = useState('');
  const [reasonError, setReasonError] = useState('');
  const requestGeneration = useRef(0);
  const requestController = useRef<AbortController | null>(null);
  const hasLoaded = useRef(false);
  const decisionInFlight = useRef(false);
  const initialFocus = useRef<HTMLElement | null>(null);
  const returnFocus = useRef<HTMLButtonElement | null>(null);
  const modal = useRef<HTMLDivElement | null>(null);
  const inboxHeading = useRef<HTMLHeadingElement | null>(null);
  const canApprove = Boolean(user?.permissions.includes('po.approve'));
  const canReject = Boolean(user?.permissions.includes('po.reject'));
  const canView = Boolean(user?.permissions.includes('po.view'));

  const load = useCallback(async () => {
    requestController.current?.abort();
    const request = ++requestGeneration.current;
    if (!canApprove) {
      setItems([]); setTotal(0); setOffset(0); setLoading(false); setLoadError(''); setHasSnapshot(false);
      return;
    }
    const controller = new AbortController();
    requestController.current = controller;
    if (!hasLoaded.current) setLoading(true);
    setLoadError('');
    try {
      const response = await fetch(`/api/purchase-order-approvals?limit=${pageLimit}&offset=${offset}`, { credentials: 'include', signal: controller.signal });
      if (!response.ok) throw new Error(await responseMessage(response));
      const payload = await response.json() as ApprovalResponse;
      if (request !== requestGeneration.current || controller.signal.aborted) return;
      const approvals = payload.items ?? [];
      const nextTotal = payload.total ?? approvals.length;
      if (offset > 0 && offset >= nextTotal) {
        setLoading(true);
        setOffset(nextTotal > 0 ? Math.floor((nextTotal - 1) / pageLimit) * pageLimit : 0);
        return;
      }
      setItems(approvals);
      setTotal(nextTotal);
      setHasSnapshot(true);
    } catch (cause) {
      if (request === requestGeneration.current && !controller.signal.aborted) {
        setLoadError(cause instanceof Error ? cause.message : 'Approvals could not be loaded');
      }
    } finally {
      if (request === requestGeneration.current) {
        hasLoaded.current = true;
        setLoading(false);
      }
    }
  }, [canApprove, offset]);

  useEffect(() => {
    void load();
    const refresh = () => void load();
    window.addEventListener(refreshEvent, refresh);
    return () => {
      window.removeEventListener(refreshEvent, refresh);
      requestController.current?.abort();
      requestGeneration.current += 1;
    };
  }, [load]);

  function changePage(nextOffset: number) {
    setLoading(true);
    setLoadError('');
    setOffset(Math.max(0, nextOffset));
  }

  function retryLoad() {
    if (!hasSnapshot) setLoading(true);
    void load();
  }

  function restoreDecisionFocus() {
    const target = returnFocus.current;
    window.setTimeout(() => target?.focus(), 0);
  }

  function closeDecision() {
    if (decisionInFlight.current) return;
    setDecision(null); setDecisionError(''); setReason(''); setReasonError('');
    restoreDecisionFocus();
  }

  function openDecision(approval: PurchaseOrderApproval, action: Decision['action'], event: MouseEvent<HTMLButtonElement>) {
    if (decisionInFlight.current) return;
    returnFocus.current = event.currentTarget;
    setDecision({ approval, action });
    setDecisionError(''); setReason(''); setReasonError('');
  }

  useEffect(() => {
    if (!decision) return;
    initialFocus.current?.focus();
    const handleKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        closeDecision();
        return;
      }
      if (event.key !== 'Tab' || !modal.current) return;
      const focusable = Array.from(modal.current.querySelectorAll<HTMLElement>('button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'));
      if (focusable.length === 0) { event.preventDefault(); return; }
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && (document.activeElement === first || !modal.current.contains(document.activeElement))) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };
    document.addEventListener('keydown', handleKey);
    return () => document.removeEventListener('keydown', handleKey);
  }, [decision]);

  async function decide(approval: PurchaseOrderApproval, action: Decision['action'], decisionReason = '') {
    if (decisionInFlight.current) return;
    decisionInFlight.current = true;
    setSavingID(approval.id); setDecisionError('');
    try {
      const response = await fetch(`/api/purchase-order-approvals/${approval.id}/${action}`, {
        method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(action === 'reject' ? { reason: decisionReason } : {})
      });
      if (!response.ok) throw new Error(await responseMessage(response));
      setItems(value => value.filter(item => item.id !== approval.id));
      setTotal(value => Math.max(0, value - 1));
      setDecision(null); setReason(''); setReasonError('');
      if (items.length === 1 && offset > 0) {
        setLoading(true);
        setOffset(Math.max(0, offset - pageLimit));
      }
      window.dispatchEvent(new CustomEvent(refreshEvent, { detail: { approvalId: approval.id } }));
      window.setTimeout(() => inboxHeading.current?.focus(), 0);
    } catch (cause) {
      setDecisionError(cause instanceof Error ? cause.message : 'Approval could not be updated');
    } finally {
      decisionInFlight.current = false;
      setSavingID('');
    }
  }

  function submitDecision(event: FormEvent) {
    event.preventDefault();
    if (!decision || decisionInFlight.current) return;
    if (decision.action === 'reject' && !reason.trim()) { setReasonError('Rejection reason is required'); return; }
    void decide(decision.approval, decision.action, reason.trim());
  }

  if (!canApprove) return <section className="module-index approval-inbox"><div className="table-empty" role="status"><strong>Approval permission is required</strong><span>You do not have access to pending purchase order approvals.</span></div></section>;

  const initialLoadFailed = !loading && !hasSnapshot && Boolean(loadError);
  const rangeStart = total === 0 ? 0 : offset + 1;
  const rangeEnd = Math.min(total, offset + items.length);

  return <section className="module-index approval-inbox">
    <div className="page-title-row"><div><h1 ref={inboxHeading} tabIndex={-1}>Approval Inbox</h1><p className="muted">Review pending supplier orders. {total} pending.</p></div></div>
    {loadError && hasSnapshot ? <div className="table-empty" role="alert"><strong>Could not refresh approvals</strong><span>{loadError}</span><button className="table-action" onClick={retryLoad}>Retry</button></div> : null}
    <div className="table-frame"><table><thead><tr><th>Purchase Order</th><th>Requested By</th><th>Requested</th><th>Actions</th></tr></thead><tbody>
      {loading ? <tr><td colSpan={4}><div className="table-empty" role="status">Loading...</div></td></tr> : initialLoadFailed ? <tr><td colSpan={4}><div className="table-empty" role="alert"><strong>Could not load approvals</strong><span>{loadError}</span><button className="table-action" onClick={retryLoad}>Retry</button></div></td></tr> : items.length === 0 ? <tr><td colSpan={4}><div className="table-empty" role="status"><strong>No purchase orders awaiting approval</strong><span>New approval requests will appear here.</span></div></td></tr> : items.map(approval => <tr key={approval.id}>
        <td><div className="approval-order-cell">{canView ? <Link aria-label={`Open ${approvalReference(approval)}`} href={`/supplier-orders/${approval.purchaseOrderId}`}>{approvalReference(approval)}</Link> : <span>{approvalReference(approval)}</span>}<small>{approval.supplierName ?? 'Supplier unavailable'}</small></div></td>
        <td>{approval.createdBy?.displayName ?? '—'}</td><td>{approval.createdAt?.slice(0, 10) || '—'}</td>
        <td><div className="approval-actions"><button className="primary-button" disabled={Boolean(savingID)} aria-label={`Approve ${approvalReference(approval)}`} onClick={event => openDecision(approval, 'approve', event)}>Approve</button>{canReject ? <button disabled={Boolean(savingID)} aria-label={`Reject ${approvalReference(approval)}`} onClick={event => openDecision(approval, 'reject', event)}>Reject</button> : null}</div></td>
      </tr>)}
    </tbody></table></div>
    {hasSnapshot ? <div className="approval-pagination" aria-label="Approval pagination"><span>{rangeStart}–{rangeEnd} of {total}</span><div><button aria-label="Previous approval page" disabled={offset === 0 || loading} onClick={() => changePage(offset - pageLimit)}>Prev</button><button aria-label="Next approval page" disabled={offset + pageLimit >= total || loading} onClick={() => changePage(offset + pageLimit)}>Next</button></div></div> : null}
    {decision ? <><button className="crud-scrim" aria-label="Close decision form" onClick={closeDecision} /><div ref={modal} className="crud-modal crud-modal--compact" role="dialog" aria-modal="true" aria-label={`${decision.action === 'approve' ? 'Approve' : 'Reject'} ${approvalReference(decision.approval)}`}><div className="crud-modal-heading"><div><strong>{decision.action === 'approve' ? 'Approve' : 'Reject'} {approvalReference(decision.approval)}</strong><span>{decision.action === 'approve' ? 'This decision cannot be undone.' : 'Provide a brief reason for the requester.'}</span></div><button aria-label="Close decision form" onClick={closeDecision} disabled={Boolean(savingID)}>×</button></div><form onSubmit={submitDecision}>{decisionError ? <p className="form-error" role="alert">{decisionError}</p> : null}{decision.action === 'reject' ? <div className="crud-fields"><label className="approval-reason"><span>Rejection reason *</span><textarea ref={element => { initialFocus.current = element; }} aria-label="Rejection reason" aria-invalid={Boolean(reasonError)} aria-describedby={reasonError ? reasonErrorID : undefined} value={reason} onChange={event => { setReason(event.target.value); setReasonError(''); }} />{reasonError ? <small id={reasonErrorID} className="form-error" role="alert">{reasonError}</small> : null}</label></div> : <div className="crud-fields"><p>Confirm approval of this purchase order.</p></div>}<div className="crud-actions"><button type="button" onClick={closeDecision} disabled={Boolean(savingID)}>Cancel</button><button ref={element => { if (decision.action === 'approve') initialFocus.current = element; }} className="primary-button" disabled={Boolean(savingID)}>{savingID ? `${decision.action === 'approve' ? 'Approving' : 'Rejecting'}...` : `${decision.action === 'approve' ? 'Approve' : 'Reject'} order`}</button></div></form></div></> : null}
  </section>;
}
