import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AppShell } from './app-shell';

const replace = vi.fn();
let currentPath = '/dashboard';
const director = { username: 'director_demo', displayName: 'Director Demo', permissions: ['po.view', 'po.price.view', 'po.approve', 'po.reject', 'inventory.view'] };

vi.mock('next/navigation', () => ({
  usePathname: () => currentPath,
  useRouter: () => ({ replace })
}));

function auth(user = { username: 'admin', displayName: 'Administrator', permissions: [] as string[] }) {
  return { ok: true, json: async () => ({ user }) } as Response;
}

function approvals(poNumber: string, supplierName: string, total = 1) {
  return { ok: true, json: async () => ({ items: [{ id: `approval-${poNumber}`, purchaseOrderId: `id-${poNumber}`, poNumber, supplierId: 'supplier-1', supplierName }], total }) } as Response;
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => { resolve = done; });
  return { promise, resolve };
}

describe('AppShell', () => {
  beforeEach(() => {
    localStorage.clear();
    currentPath = '/dashboard';
    replace.mockClear();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(auth()));
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

  it('uses enriched approvals and API total with real Director permissions', async () => {
    const fetchMock = vi.fn((path: string) => Promise.resolve(path === '/api/auth/me' ? auth(director) : approvals('PO-202607-00001', 'Acme', 205)));
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);

    expect(await screen.findByText('Director Demo')).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId('notification-badge')).toHaveTextContent('205'));
    await user.click(screen.getByRole('button', { name: 'Notifications' }));
    expect(screen.getByRole('link', { name: /PO-202607-00001 awaits approval/ })).toHaveAttribute('href', '/supplier-orders/id-PO-202607-00001');
    expect(screen.getByText('Acme')).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith('/api/purchase-order-approvals?limit=200', expect.objectContaining({ signal: expect.any(AbortSignal) }));
    expect(fetchMock.mock.calls.filter(([path]) => path === '/api/purchase-order-approvals?limit=200').length).toBeGreaterThan(0);
    expect(fetchMock.mock.calls.every(([path]) => path === '/api/auth/me' || path === '/api/purchase-order-approvals?limit=200')).toBe(true);
  });

  it('ignores and aborts stale notification refreshes', async () => {
    const stale = deferred<Response>();
    const signals: AbortSignal[] = [];
    let approvalRequest = 0;
    vi.stubGlobal('fetch', vi.fn((path: string, init?: RequestInit) => {
      if (path === '/api/auth/me') return Promise.resolve(auth(director));
      signals.push(init?.signal as AbortSignal);
      approvalRequest += 1;
      if (approvalRequest === 1) return Promise.resolve(approvals('PO-INITIAL', 'Initial'));
      if (approvalRequest === 2) return stale.promise;
      return Promise.resolve(approvals('PO-NEW', 'Newest'));
    }));
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);
    await waitFor(() => expect(screen.getByTestId('notification-badge')).toHaveTextContent('1'));

    window.dispatchEvent(new Event('purchase-order-approvals:refresh'));
    window.dispatchEvent(new Event('purchase-order-approvals:refresh'));
    expect(signals[1].aborted).toBe(true);
    await user.click(screen.getByRole('button', { name: 'Notifications' }));
    expect(await screen.findByText('PO-NEW awaits approval')).toBeInTheDocument();

    stale.resolve(approvals('PO-STALE', 'Stale'));
    await waitFor(() => expect(screen.queryByText('PO-STALE awaits approval')).not.toBeInTheDocument());
    expect(screen.getByText('PO-NEW awaits approval')).toBeInTheDocument();
  });

  it('preserves the last good notification snapshot when refresh fails', async () => {
    let approvalRequest = 0;
    const fetchMock = vi.fn((path: string) => {
      if (path === '/api/auth/me') return Promise.resolve(auth(director));
      approvalRequest += 1;
      return Promise.resolve(approvalRequest === 1 ? approvals('PO-GOOD', 'Good Supplier', 4) : { ok: false } as Response);
    });
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);
    await waitFor(() => expect(screen.getByTestId('notification-badge')).toHaveTextContent('4'));

    window.dispatchEvent(new Event('purchase-order-approvals:refresh'));
    await waitFor(() => expect(fetchMock.mock.calls.filter(([path]) => path === '/api/purchase-order-approvals?limit=200').length).toBeGreaterThan(1));
    await user.click(screen.getByRole('button', { name: 'Notifications' }));

    expect(screen.getByText('PO-GOOD awaits approval')).toBeInTheDocument();
    expect(screen.getByTestId('notification-badge')).toHaveTextContent('4');
  });
});
