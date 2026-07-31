import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useEffect } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AppShell } from './app-shell';

const replace = vi.fn();
const router = { replace };
let currentPath = '/dashboard';
const adminPermissions = ['dashboard.view', 'master_data.view', 'po.view', 'po.create', 'po.approve', 'inventory.view', 'receiving.view', 'user.manage', 'role.manage', 'smtp_settings.view', 'email_log.view'];
const director = { username: 'director_demo', displayName: 'Director Demo', permissions: ['po.view', 'po.price.view', 'po.approve', 'po.reject', 'inventory.view'] };

vi.mock('next/navigation', () => ({
  usePathname: () => currentPath,
  useRouter: () => router
}));

function auth(user = { username: 'admin', displayName: 'Administrator', permissions: adminPermissions }) {
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

  it('shows only permitted links and removes empty navigation groups', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(auth({ username: 'roles', displayName: 'Role Manager', permissions: ['role.manage'] })));
    currentPath = '/settings/roles';
    render(<AppShell title="Roles"><div>roles content</div></AppShell>);

    await screen.findByText('Role Manager');
    expect(screen.getByRole('link', { name: 'Roles & Permissions' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Users' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Dashboard' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Data Master' })).not.toBeInTheDocument();
  });

  it('does not mount unauthorized route content and offers the first permitted module', async () => {
    const mounted = vi.fn();
    function ProtectedChild() {
      useEffect(() => { mounted(); }, []);
      return <div>protected inventory</div>;
    }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(auth({ username: 'roles', displayName: 'Role Manager', permissions: ['role.manage'] })));
    currentPath = '/inventory';
    render(<AppShell title="Inventory"><ProtectedChild /></AppShell>);

    expect(await screen.findByRole('heading', { name: 'Access Denied' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to an available module' })).toHaveAttribute('href', '/settings/roles');
    expect(screen.queryByText('protected inventory')).not.toBeInTheDocument();
    expect(mounted).not.toHaveBeenCalled();
  });

  it('orders master data first and keeps approval and delivery notes out of the sidebar', async () => {
    currentPath = '/inventory';
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);
    await screen.findByText('Administrator');
    const sidebar = screen.getByLabelText('Main navigation');
    const text = sidebar.textContent ?? '';

    expect(text.indexOf('Data Master')).toBeLessThan(text.indexOf('Procurement'));
    expect(text.indexOf('Procurement')).toBeLessThan(text.indexOf('Logistics'));
    expect(text.indexOf('Logistics')).toBeLessThan(text.indexOf('Reports'));
    expect(text.indexOf('Stock Inventory')).toBeLessThan(text.indexOf('Receiving'));
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
    currentPath = '/units';
    render(<AppShell title="Units"><div>content</div></AppShell>);

    expect(await screen.findByText('Administrator')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Measurements' })).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByRole('link', { name: 'Unit' })).toHaveAttribute('href', '/units');
    expect(screen.getByRole('link', { name: 'Unit' })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('link', { name: 'Category' })).toHaveAttribute('href', '/categories');
    expect(screen.getByRole('link', { name: 'Packing' })).toHaveAttribute('href', '/packings');
    expect(screen.getByRole('link', { name: 'Plants' })).toHaveAttribute('href', '/plants');
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

  it('opens the user menu with the authenticated username', async () => {
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);

    await screen.findByText('Administrator');
    const trigger = screen.getByRole('button', { name: 'Open user menu' });
    expect(trigger.querySelector('svg.dropdown-chevron')).toBeInTheDocument();
    expect(trigger).not.toHaveTextContent('⌄');
    await user.click(trigger);

    expect(screen.getByRole('dialog', { name: 'User menu' })).toBeInTheDocument();
    expect(trigger).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText('@admin')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Logout' })).toHaveFocus();
  });

  it('posts logout with credentials and replaces history after a 204 response', async () => {
    const fetchMock = vi.fn((path: string) => Promise.resolve(path === '/api/auth/me' ? auth() : { status: 204 } as Response));
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);

    await screen.findByText('Administrator');
    await user.click(screen.getByRole('button', { name: 'Open user menu' }));
    await user.click(screen.getByRole('button', { name: 'Logout' }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/auth/logout', { method: 'POST', credentials: 'include' }));
    await waitFor(() => expect(replace).toHaveBeenCalledWith('/login'));
    expect(screen.queryByRole('dialog', { name: 'User menu' })).not.toBeInTheDocument();
  });

  it('closes the user menu on Escape and outside pointer interaction', async () => {
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);

    await screen.findByText('Administrator');
    const trigger = screen.getByRole('button', { name: 'Open user menu' });
    await user.click(trigger);
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('dialog', { name: 'User menu' })).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();

    await user.click(trigger);
    await user.pointer({ target: document.body, keys: '[MouseLeft]' });
    expect(screen.queryByRole('dialog', { name: 'User menu' })).not.toBeInTheDocument();
  });

  it('keeps the user menu and session UI visible when logout fails', async () => {
    const fetchMock = vi.fn((path: string) => Promise.resolve(path === '/api/auth/me' ? auth() : { status: 500 } as Response));
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);

    await screen.findByText('Administrator');
    await user.click(screen.getByRole('button', { name: 'Open user menu' }));
    await user.click(screen.getByRole('button', { name: 'Logout' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Unable to log out. Please try again.');
    expect(screen.getByRole('dialog', { name: 'User menu' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Open user menu' })).toBeInTheDocument();
    expect(replace).not.toHaveBeenCalled();
  });

  it('keeps a pending logout menu open through Escape and outside interaction, then announces the failure', async () => {
    const pending = deferred<Response>();
    const fetchMock = vi.fn((path: string) => path === '/api/auth/me' ? Promise.resolve(auth()) : pending.promise);
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);

    await screen.findByText('Administrator');
    await user.click(screen.getByRole('button', { name: 'Open user menu' }));
    await user.click(screen.getByRole('button', { name: 'Logout' }));
    expect(screen.getByRole('button', { name: 'Logging out...' })).toBeDisabled();

    await user.keyboard('{Escape}');
    await user.pointer({ target: document.body, keys: '[MouseLeft]' });
    expect(screen.getByRole('dialog', { name: 'User menu' })).toBeInTheDocument();

    pending.resolve({ status: 500 } as Response);
    expect(await screen.findByRole('alert')).toHaveTextContent('Unable to log out. Please try again.');
    expect(screen.getByRole('dialog', { name: 'User menu' })).toBeInTheDocument();
  });

  it('prevents duplicate logout requests and redirects after a successful retry', async () => {
    const firstAttempt = deferred<Response>();
    const fetchMock = vi.fn((path: string) => {
      if (path === '/api/auth/me') return Promise.resolve(auth());
      return fetchMock.mock.calls.filter(([requestPath]) => requestPath === '/api/auth/logout').length === 1
        ? firstAttempt.promise
        : Promise.resolve({ status: 204 } as Response);
    });
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);

    await screen.findByText('Administrator');
    await user.click(screen.getByRole('button', { name: 'Open user menu' }));
    const logoutButton = screen.getByRole('button', { name: 'Logout' });
    await user.click(logoutButton);
    await user.click(logoutButton);
    expect(fetchMock.mock.calls.filter(([path]) => path === '/api/auth/logout')).toHaveLength(1);

    firstAttempt.resolve({ status: 500 } as Response);
    expect(await screen.findByRole('alert')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Logout' }));

    await waitFor(() => expect(fetchMock.mock.calls.filter(([path]) => path === '/api/auth/logout')).toHaveLength(2));
    await waitFor(() => expect(replace).toHaveBeenCalledWith('/login'));
  });

  it('keeps only one header popover open at a time', async () => {
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);

    await screen.findByText('Administrator');
    await user.click(screen.getByRole('button', { name: 'Notifications' }));
    expect(screen.getByRole('link', { name: /View all notifications/ })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Open user menu' }));
    expect(screen.getByRole('dialog', { name: 'User menu' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: /View all notifications/ })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Notifications' }));
    expect(screen.queryByRole('dialog', { name: 'User menu' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: /View all notifications/ })).toBeInTheDocument();
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

  it('optimistically removes a decided approval and never restores it when refresh fails', async () => {
    let approvalRequest = 0;
    const fetchMock = vi.fn((path: string) => {
      if (path === '/api/auth/me') return Promise.resolve(auth(director));
      approvalRequest += 1;
      return Promise.resolve(approvalRequest === 1
        ? { ok: true, json: async () => ({ items: [
          { id: 'approval-decided', purchaseOrderId: 'po-decided', poNumber: 'PO-DECIDED', supplierName: 'Supplier A' },
          { id: 'approval-other', purchaseOrderId: 'po-other', poNumber: 'PO-OTHER', supplierName: 'Supplier B' }
        ], total: 2 }) } as Response
        : { ok: false } as Response);
    });
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);
    await waitFor(() => expect(screen.getByTestId('notification-badge')).toHaveTextContent('2'));

    window.dispatchEvent(new CustomEvent('purchase-order-approvals:refresh', { detail: { approvalId: 'approval-decided' } }));
    await waitFor(() => expect(fetchMock.mock.calls.filter(([path]) => path === '/api/purchase-order-approvals?limit=200').length).toBe(2));
    await user.click(screen.getByRole('button', { name: 'Notifications' }));

    expect(screen.queryByText('PO-DECIDED awaits approval')).not.toBeInTheDocument();
    expect(screen.getByText('PO-OTHER awaits approval')).toBeInTheDocument();
    expect(screen.getByTestId('notification-badge')).toHaveTextContent('1');
  });
});
