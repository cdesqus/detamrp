import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApprovalInbox, PurchaseOrderApproval } from './approval-inbox';

const directorPermissions = ['po.view', 'po.price.view', 'po.approve', 'po.reject', 'inventory.view'];
const currentUser = { username: 'director_demo', displayName: 'Director Demo', permissions: directorPermissions };

vi.mock('../app-shell/app-shell', () => ({ useCurrentUser: () => currentUser }));

function approval(overrides: Partial<PurchaseOrderApproval> = {}): PurchaseOrderApproval {
  return {
    id: 'approval-1', purchaseOrderId: 'po-1', poNumber: 'PO-202607-00001', supplierId: 'supplier-1', supplierName: 'Acme', version: 1,
    status: 'PENDING', createdBy: { displayName: 'Buyer Demo' }, createdAt: '2026-07-21T08:00:00Z', ...overrides
  };
}

function ok(items: PurchaseOrderApproval[], total = items.length) {
  return { ok: true, json: async () => ({ items, total }) } as Response;
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => { resolve = done; });
  return { promise, resolve };
}

describe('ApprovalInbox', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(ok([approval()])));
  });

  it('uses the enriched approval response with real Director permissions and API total', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(ok([approval()], 205)));
    render(<ApprovalInbox />);

    expect(await screen.findByText('PO-202607-00001')).toBeInTheDocument();
    expect(screen.getByText('Acme')).toBeInTheDocument();
    expect(screen.getByText(/205 pending/)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Open PO-202607-00001' })).toHaveAttribute('href', '/supplier-orders/po-1');
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith('/api/purchase-order-approvals?limit=200', expect.objectContaining({ credentials: 'include' }));
    expect(fetch).not.toHaveBeenCalledWith(expect.stringContaining('/api/master-data/'), expect.anything());
  });

  it('ignores a stale refresh, aborts it, and keeps the newest snapshot', async () => {
    const stale = deferred<Response>();
    const signals: AbortSignal[] = [];
    let approvalRequest = 0;
    vi.stubGlobal('fetch', vi.fn((_path: string, init?: RequestInit) => {
      signals.push(init?.signal as AbortSignal);
      approvalRequest += 1;
      if (approvalRequest === 1) return Promise.resolve(ok([approval()]));
      if (approvalRequest === 2) return stale.promise;
      return Promise.resolve(ok([approval({ id: 'approval-new', poNumber: 'PO-NEW' })]));
    }));
    render(<ApprovalInbox />);
    await screen.findByText('PO-202607-00001');

    window.dispatchEvent(new Event('purchase-order-approvals:refresh'));
    window.dispatchEvent(new Event('purchase-order-approvals:refresh'));
    expect(await screen.findByText('PO-NEW')).toBeInTheDocument();
    expect(signals[1].aborted).toBe(true);

    stale.resolve(ok([approval({ id: 'approval-stale', poNumber: 'PO-STALE' })]));
    await waitFor(() => expect(screen.queryByText('PO-STALE')).not.toBeInTheDocument());
    expect(screen.getByText('PO-NEW')).toBeInTheDocument();
  });

  it('preserves the last good inbox snapshot when refresh fails', async () => {
    let approvalRequest = 0;
    vi.stubGlobal('fetch', vi.fn(() => {
      approvalRequest += 1;
      return Promise.resolve(approvalRequest === 1
        ? ok([approval()], 3)
        : { ok: false, json: async () => ({ message: 'Approvals are temporarily unavailable' }) } as Response);
    }));
    render(<ApprovalInbox />);
    await screen.findByText('PO-202607-00001');

    window.dispatchEvent(new Event('purchase-order-approvals:refresh'));

    expect(await screen.findByRole('alert')).toHaveTextContent('Approvals are temporarily unavailable');
    expect(screen.getByText('PO-202607-00001')).toBeInTheDocument();
    expect(screen.getByText(/3 pending/)).toBeInTheDocument();
  });

  it('posts one approve decision with an empty JSON body', async () => {
    const decision = deferred<Response>();
    const fetchMock = vi.fn((path: string) => path.includes('/approve') ? decision.promise : Promise.resolve(ok([approval()])));
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    render(<ApprovalInbox />);

    const trigger = await screen.findByRole('button', { name: 'Approve PO-202607-00001' });
    await user.click(trigger);
    const submit = screen.getByRole('button', { name: 'Approve order' });
    expect(submit).toHaveFocus();
    await user.dblClick(submit);

    expect(fetchMock).toHaveBeenCalledWith('/api/purchase-order-approvals/approval-1/approve', expect.objectContaining({ method: 'POST', body: '{}' }));
    expect(fetchMock.mock.calls.filter(([path]) => String(path).includes('/approve'))).toHaveLength(1);
    expect(submit).toBeDisabled();
    decision.resolve({ ok: true, json: async () => ({ ...approval(), status: 'APPROVED' }) } as Response);
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('supports Escape, restores focus, and exposes reject validation accessibly', async () => {
    const user = userEvent.setup();
    render(<ApprovalInbox />);

    const trigger = await screen.findByRole('button', { name: 'Reject PO-202607-00001' });
    await user.click(trigger);
    const reason = screen.getByRole('textbox', { name: 'Rejection reason' });
    expect(reason).toHaveFocus();
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    await waitFor(() => expect(trigger).toHaveFocus());

    await user.click(trigger);
    await user.click(screen.getByRole('button', { name: 'Reject order' }));
    const error = screen.getByRole('alert');
    const invalidReason = screen.getByRole('textbox', { name: 'Rejection reason' });
    expect(error).toHaveTextContent('Rejection reason is required');
    expect(invalidReason).toHaveAttribute('aria-invalid', 'true');
    expect(invalidReason).toHaveAttribute('aria-describedby', error.id);
  });

  it('posts the trimmed reject reason to the reject endpoint', async () => {
    const fetchMock = vi.fn((path: string) => path.includes('/reject')
      ? Promise.resolve({ ok: true, json: async () => ({ ...approval(), status: 'REJECTED' }) } as Response)
      : Promise.resolve(ok([approval()])));
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    render(<ApprovalInbox />);

    await user.click(await screen.findByRole('button', { name: 'Reject PO-202607-00001' }));
    await user.type(screen.getByRole('textbox', { name: 'Rejection reason' }), '  Budget is unavailable  ');
    await user.click(screen.getByRole('button', { name: 'Reject order' }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/purchase-order-approvals/approval-1/reject', expect.objectContaining({
      method: 'POST', body: JSON.stringify({ reason: 'Budget is unavailable' })
    })));
  });
});
