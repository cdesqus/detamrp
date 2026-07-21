import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AppShell } from './app-shell';

const replace = vi.fn();
let currentPath = '/dashboard';

vi.mock('next/navigation', () => ({
  usePathname: () => currentPath,
  useRouter: () => ({ replace })
}));

describe('AppShell', () => {
  beforeEach(() => {
    localStorage.clear();
    currentPath = '/dashboard';
    replace.mockClear();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ user: { username: 'admin', displayName: 'Administrator', permissions: [] } })
    }));
  });

  it('orders master data first and keeps approval and delivery notes out of the sidebar', async () => {
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);
    await screen.findByText('Administrator');
    const sidebar = screen.getByLabelText('Main navigation');
    const text = sidebar.textContent ?? '';

    expect(text.indexOf('Data Master')).toBeLessThan(text.indexOf('Procurement'));
    expect(text.indexOf('Procurement')).toBeLessThan(text.indexOf('Logistics'));
    expect(text.indexOf('Logistics')).toBeLessThan(text.indexOf('Reports'));
    expect(screen.queryByRole('link', { name: 'Delivery Notes' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Approval Inbox' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Dashboard' }).querySelector('svg')).toBeInTheDocument();
  });

  it('owns the desktop collapse control in the sidebar and expands active settings', async () => {
    currentPath = '/settings/users';
    render(<AppShell title="Users"><div>content</div></AppShell>);
    await screen.findByText('Administrator');
    const sidebar = screen.getByLabelText('Main navigation');
    const header = document.querySelector('.app-content > header');

    expect(within(sidebar).getByRole('button', { name: 'Collapse sidebar' })).toBeInTheDocument();
    expect(within(header as HTMLElement).queryByRole('button', { name: 'Collapse sidebar' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Users' })).toHaveAttribute('aria-current', 'page');
  });

  it('renders live navigation and marks the current route', async () => {
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);

    expect(await screen.findByText('Administrator')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Dashboard' })).toHaveAttribute('aria-current', 'page');
    await user.click(screen.getByRole('button', { name: 'Data Master' }));
    expect(screen.getByRole('link', { name: 'Raw Materials' })).toHaveAttribute('href', '/raw-materials');
    expect(screen.getByRole('link', { name: 'Raw Materials' })).toHaveClass('nav-child-link');
  });

  it('collapses the sidebar and persists the preference', async () => {
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);
    await screen.findByText('Administrator');

    await user.click(screen.getByRole('button', { name: 'Collapse sidebar' }));

    await waitFor(() => expect(localStorage.getItem('order-stock.sidebar-collapsed')).toBe('true'));
    expect(screen.getByRole('button', { name: 'Expand sidebar' })).toBeInTheDocument();
  });

  it('loads approval notifications for approvers and refreshes them after a decision event', async () => {
    let refreshed = false;
    vi.stubGlobal('fetch', vi.fn((path: string) => Promise.resolve(path === '/api/auth/me'
      ? { ok: true, json: async () => ({ user: { username: 'director', displayName: 'Director', permissions: ['po.approve', 'po.view', 'master_data.view'] } }) }
      : path === '/api/purchase-order-approvals' ? { ok: true, json: async () => ({ items: refreshed ? [] : [{ id: 'approval-1', purchaseOrderId: 'po-1' }], total: refreshed ? 0 : 1 }) }
        : path === '/api/purchase-orders/po-1' ? { ok: true, json: async () => ({ id: 'po-1', poNumber: 'PO-202607-00001', supplierId: 'supplier-1' }) }
          : { ok: true, json: async () => ({ items: [{ id: 'supplier-1', code: 'SUP-001', name: 'Acme' }] }) })));
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);

    expect(await screen.findByText('Director')).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId('notification-badge')).toHaveTextContent('1'));
    await user.click(screen.getByRole('button', { name: 'Notifications' }));
    expect(screen.getByRole('link', { name: /PO-202607-00001 awaits approval/ })).toHaveAttribute('href', '/supplier-orders/po-1');
    expect(screen.getByText('SUP-001 — Acme')).toBeInTheDocument();

    refreshed = true;
    window.dispatchEvent(new Event('purchase-order-approvals:refresh'));
    await waitFor(() => expect(screen.queryByTestId('notification-badge')).not.toBeInTheDocument());
  });
});
