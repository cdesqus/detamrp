import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, it, vi } from 'vitest';
import { ActivityLog } from './activity-log';

const event = {
  id: 'f8eb2c5e-1a8b-44b9-b85a-2e67fe8b9710',
  occurredAt: '2026-07-30T09:15:00Z',
  actorUserId: '8c8fc08a-c496-4508-9e02-adcdb988ab3a',
  actorName: 'Dizey',
  module: 'PROCUREMENT',
  action: 'APPROVED',
  targetType: 'purchase_orders',
  targetCode: 'PO-202607-00004',
  before: { status: 'PENDING_APPROVAL', total_amount: 150000 },
  after: { status: 'APPROVED', total_amount: 150000 }
};

function payload(items = [event]) {
  return {
    items,
    total: items.length,
    page: 1,
    pageSize: 20,
    filters: {
      actors: [{ id: event.actorUserId, name: event.actorName }],
      modules: ['PROCUREMENT', 'INVENTORY'],
      actions: ['APPROVED', 'MOVED']
    }
  };
}

it('loads, filters, paginates, and opens structured change details', async () => {
  const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify(payload()))));
  vi.stubGlobal('fetch', fetchMock);
  const user = userEvent.setup();
  render(<ActivityLog />);

  expect(await screen.findByText('PO-202607-00004')).toBeInTheDocument();
  expect(screen.getAllByText('Dizey').length).toBeGreaterThan(0);

  await user.selectOptions(screen.getByRole('combobox', { name: 'Module' }), 'INVENTORY');
  await waitFor(() => {
    const lastURL = String(fetchMock.mock.calls.at(-1)?.[0]);
    expect(lastURL).toContain('module=INVENTORY');
    expect(lastURL).toContain('pageSize=20');
  });

  await user.click(screen.getByRole('button', { name: 'View details for PO-202607-00004' }));
  expect(screen.getByRole('dialog', { name: 'Activity details' })).toBeInTheDocument();
  expect(screen.getByText('PENDING_APPROVAL')).toBeInTheDocument();
  expect(screen.getByText('PENDING_APPROVAL').closest('table')?.parentElement).toHaveClass('table-detail');
  expect(screen.getAllByText('APPROVED').length).toBeGreaterThan(0);
});

it('renders an empty state', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(payload([])))));
  render(<ActivityLog />);

  expect(await screen.findByText('No activity matches these filters.')).toBeInTheDocument();
});

it('renders a useful load error', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 500 })));
  render(<ActivityLog />);

  expect(await screen.findByRole('alert')).toHaveTextContent('Activity log could not be loaded.');
});
