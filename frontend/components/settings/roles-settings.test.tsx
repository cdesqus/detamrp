import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, expect, it, vi } from 'vitest';
import { RolesSettings } from './roles-settings';

beforeEach(() => vi.stubGlobal('fetch', vi.fn(input => {
  const data = String(input).includes('permissions') ? { items: [
    { code: 'dashboard.view', description: 'View dashboard', group: 'Dashboard' },
    { code: 'master_data.view', description: 'View master data', group: 'Data Master' },
    { code: 'po.approve', description: 'Approve supplier orders', group: 'Procurement' }
  ] } : { items: [], total: 0 };
  return Promise.resolve({ ok: true, json: async () => data });
})));

it('opens a role editor with permissions grouped by navigation module', async () => {
  render(<RolesSettings />);
  const user = userEvent.setup();
  await user.click(await screen.findByRole('button', { name: 'New role' }));

  expect(screen.getByRole('dialog', { name: 'New role' })).toBeInTheDocument();
  expect(screen.getByText('Dashboard')).toBeInTheDocument();
  expect(screen.getByText('Data Master')).toBeInTheDocument();
  expect(screen.getByText('Procurement')).toBeInTheDocument();
  expect(screen.getByLabelText('View dashboard')).toBeInTheDocument();
});
