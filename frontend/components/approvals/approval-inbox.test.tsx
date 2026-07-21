import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApprovalInbox } from './approval-inbox';

const approval = {
  id: 'approval-1', tenantId: 'tenant-1', purchaseOrderId: 'po-1', version: 1,
  approverUserId: 'user-1', approverDisplayName: 'Director Demo', approverEmail: 'director@example.com',
  status: 'PENDING', decisionReason: '', decidedAt: null, decidedByUserId: '',
  createdBy: { displayName: 'Buyer Demo' }, createdAt: '2026-07-21T08:00:00Z',
  updatedBy: { displayName: 'Buyer Demo' }, updatedAt: '2026-07-21T08:00:00Z'
};

const currentUser = { username: 'director_demo', displayName: 'Director Demo', permissions: ['po.approve', 'po.reject', 'po.view', 'master_data.view'] };

vi.mock('../app-shell/app-shell', () => ({ useCurrentUser: () => currentUser }));

describe('ApprovalInbox', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ items: [approval], total: 1 }) }));
  });

  it('loads pending approvals with their PO and supplier details', async () => {
    vi.stubGlobal('fetch', vi.fn((path: string) => Promise.resolve(path === '/api/purchase-order-approvals'
      ? { ok: true, json: async () => ({ items: [approval], total: 1 }) }
      : path === '/api/purchase-orders/po-1'
        ? { ok: true, json: async () => ({ id: 'po-1', poNumber: 'PO-202607-00001', supplierId: 'supplier-1' }) }
        : { ok: true, json: async () => ({ items: [{ id: 'supplier-1', code: 'SUP-001', name: 'Acme' }] }) })));
    render(<ApprovalInbox />);

    expect(await screen.findByText('PO-202607-00001')).toBeInTheDocument();
    expect(screen.getByText('SUP-001 — Acme')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Open PO-202607-00001' })).toHaveAttribute('href', '/supplier-orders/po-1');
    expect(fetch).toHaveBeenCalledWith('/api/purchase-order-approvals', { credentials: 'include' });
  });

  it('shows a compact empty state when no approvals are pending', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ items: [], total: 0 }) }));
    render(<ApprovalInbox />);
    expect(await screen.findByText('No purchase orders awaiting approval')).toBeInTheDocument();
  });

  it('confirms approval, posts the real decision endpoint, and refreshes notifications', async () => {
    const user = userEvent.setup();
    const dispatch = vi.spyOn(window, 'dispatchEvent');
    let decided = false;
    vi.stubGlobal('fetch', vi.fn((path: string) => Promise.resolve(path === '/api/purchase-order-approvals'
      ? { ok: true, json: async () => ({ items: decided ? [] : [approval], total: decided ? 0 : 1 }) }
      : path === '/api/purchase-orders/po-1' ? { ok: false, json: async () => ({}) }
        : (decided = true, { ok: true, json: async () => ({ ...approval, status: 'APPROVED' }) }))));
    render(<ApprovalInbox />);

    await user.click(await screen.findByRole('button', { name: 'Approve PO po-1' }));
    expect(screen.getByRole('dialog', { name: 'Approve PO po-1' })).toHaveClass('crud-modal');
    await user.click(screen.getByRole('button', { name: 'Approve order' }));
    await waitFor(() => expect(fetch).toHaveBeenCalledWith('/api/purchase-order-approvals/approval-1/approve', expect.objectContaining({ method: 'POST', body: '{}' })));
    expect(dispatch).toHaveBeenCalledWith(expect.objectContaining({ type: 'purchase-order-approvals:refresh' }));
    expect(screen.queryByText('PO po-1')).not.toBeInTheDocument();
  });

  it('requires a reject reason in a centered dialog and displays a backend conflict', async () => {
    const user = userEvent.setup();
    vi.stubGlobal('fetch', vi.fn((path: string) => Promise.resolve(path === '/api/purchase-order-approvals'
      ? { ok: true, json: async () => ({ items: [approval], total: 1 }) }
      : path === '/api/purchase-orders/po-1' ? { ok: false, json: async () => ({}) }
        : { ok: false, status: 409, json: async () => ({ message: 'Approval has already been decided' }) })));
    render(<ApprovalInbox />);

    await user.click(await screen.findByRole('button', { name: 'Reject PO po-1' }));
    expect(screen.getByRole('dialog', { name: 'Reject PO po-1' })).toHaveClass('crud-modal');
    await user.click(screen.getByRole('button', { name: 'Reject order' }));
    expect(screen.getByText('Rejection reason is required')).toBeInTheDocument();

    await user.type(screen.getByRole('textbox', { name: 'Rejection reason' }), 'Budget is unavailable');
    await user.click(screen.getByRole('button', { name: 'Reject order' }));
    expect(await screen.findByRole('dialog', { name: 'Reject PO po-1' })).toHaveTextContent('Approval has already been decided');
  });
});
