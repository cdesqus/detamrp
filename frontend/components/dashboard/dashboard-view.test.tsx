import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { DashboardView } from './dashboard-view';

const replace = vi.fn();
vi.mock('next/navigation', () => ({
  useRouter: () => ({ replace }),
  useSearchParams: () => new URLSearchParams(),
}));

const snapshot = {
  filter: { from: '2026-06-25T00:00:00Z', to: '2026-07-24T00:00:00Z' },
  metrics: { pendingApproval: 2, openPO: 3, receivedKanban: 7, outstandingKanban: 4, currentStock: 5 },
  trend: [{ date: '2026-07-24', ordered: 5, received: 3 }],
  poStatus: [{ status: 'APPROVED', count: 3 }],
  outstandingBySupplier: [{ supplier: 'PT Example', kanban: 4 }],
  activities: [{ id: '1', type: 'PO', label: 'PO-001 · Approved', occurredAt: '2026-07-24T09:00:00Z' }],
};

describe('DashboardView', () => {
  beforeEach(() => {
    replace.mockReset();
    vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/master-data/suppliers')) {
        return Promise.resolve(new Response(JSON.stringify({ items: [{ id: 'supplier-1', code: 'SUP', name: 'PT Example' }] }), { status: 200 }));
      }
      return Promise.resolve(new Response(JSON.stringify(snapshot), { status: 200 }));
    }));
  });

  it('renders live metrics and activity from the API', async () => {
    render(<DashboardView />);
    expect(await screen.findByText('PO-001 · Approved')).toBeInTheDocument();
    expect(screen.getByText('Pending Approval').nextElementSibling).toHaveTextContent('2');
    expect(screen.getByText('Current Stock').nextElementSibling).toHaveTextContent('5');
    expect(screen.getByLabelText('PO and receiving trend')).toBeInTheDocument();
  });

  it('applies filters through the URL and resets them', async () => {
    render(<DashboardView />);
    await screen.findByRole('option', { name: 'SUP — PT Example' });
    fireEvent.change(screen.getByLabelText('Supplier'), { target: { value: 'supplier-1' } });
    fireEvent.click(screen.getByRole('button', { name: 'Apply' }));
    expect(replace).toHaveBeenCalledWith(expect.stringContaining('supplierId=supplier-1'));

    fireEvent.click(screen.getByRole('button', { name: 'Reset' }));
    expect(replace).toHaveBeenLastCalledWith('/dashboard');
  });

  it('shows retry when the dashboard request fails', async () => {
    vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
      if (String(input).includes('/master-data/suppliers')) return Promise.resolve(new Response(JSON.stringify({ items: [] }), { status: 200 }));
      return Promise.resolve(new Response('{}', { status: 500 }));
    }));
    render(<DashboardView />);
    expect(await screen.findByRole('alert')).toHaveTextContent('Dashboard could not be loaded.');
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
    await waitFor(() => expect(fetch).toHaveBeenCalled());
  });
});
