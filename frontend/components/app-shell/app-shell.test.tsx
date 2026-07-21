import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AppShell } from './app-shell';

const replace = vi.fn();

vi.mock('next/navigation', () => ({
  usePathname: () => '/dashboard',
  useRouter: () => ({ replace })
}));

describe('AppShell', () => {
  beforeEach(() => {
    localStorage.clear();
    replace.mockClear();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ user: { username: 'admin', displayName: 'Administrator', permissions: [] } })
    }));
  });

  it('renders live navigation and marks the current route', async () => {
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);

    expect(await screen.findByText('Administrator')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Dashboard' })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('link', { name: 'Raw Materials' })).toHaveAttribute('href', '/raw-materials');
  });

  it('collapses the sidebar and persists the preference', async () => {
    const user = userEvent.setup();
    render(<AppShell title="Dashboard"><div>content</div></AppShell>);
    await screen.findByText('Administrator');

    await user.click(screen.getByRole('button', { name: 'Collapse sidebar' }));

    await waitFor(() => expect(localStorage.getItem('order-stock.sidebar-collapsed')).toBe('true'));
    expect(screen.getByRole('button', { name: 'Expand sidebar' })).toBeInTheDocument();
  });
});
