import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { ProductionDashboardView } from './dashboard-pages';

const { auth } = vi.hoisted(() => ({ auth: { permissions: ['production.view', 'production.report'] } }));
vi.mock('next/navigation', () => ({ useRouter: () => ({ push: vi.fn() }) }));
vi.mock('../app-shell/app-shell', () => ({ useCurrentUser: () => auth }));

const dashboard = {
  filter: { from: '2026-09-01', to: '2026-09-15' },
  totals: {
    plannedQty: '100', processedQty: '50', goodQty: '19', rejectQty: '3', rejectRate: '6', achievement: '19',
    wipQty: '8', wipValue: '150', materialActual: '300', processActual: '310', totalActual: '610',
    entries: 3, activeOrders: 1, openOrders: 1, completedOrders: 0, currency: 'IDR',
  },
  parts: [
    { partNumber: 'FG-001', partName: 'Panel', unitCode: 'PCS', orders: 1, plannedQty: '100', goodQty: '19', rejectQty: '3', achievement: '19' },
  ],
  processes: [
    { code: 'STAMPING', quantity: '3', value: '56.25' },
    { code: 'WELDING', quantity: '5', value: '93.75' },
  ],
  quality: [
    { code: 'STAMPING', processedQty: '30', goodQty: '28', rejectQty: '2', rejectRate: '6.67' },
    { code: 'WELDING', processedQty: '20', goodQty: '19', rejectQty: '1', rejectRate: '5' },
  ],
  costs: [
    { partNumber: 'FG-001', partName: 'Panel', orders: 1, goodQty: '19', materialActual: '300', processActual: '310', totalActual: '610', costPerPiece: '32.1', currency: 'IDR' },
  ],
  generatedAt: '2026-09-15T08:00:00Z',
};
const respond = (value: unknown) => ({ ok: true, json: async () => value });

beforeEach(() => {
  vi.clearAllMocks();
  auth.permissions = ['production.view', 'production.report'];
});

it('shows plan attainment, WIP per process and reject rate', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond(dashboard)));
  render(<ProductionDashboardView />);
  expect(await screen.findByLabelText('FG-001: 19 good of 100 planned PCS')).toBeInTheDocument();
  expect(screen.getByLabelText('STAMPING: 3 in progress')).toBeInTheDocument();
  expect(screen.getByLabelText('WELDING: 5% reject rate of 20 processed')).toBeInTheDocument();
});

it('reads the same figures as a table for every chart', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond(dashboard)));
  render(<ProductionDashboardView />);
  await screen.findByText('Output per part');
  const row = screen.getAllByText('Panel')[0].closest('tr')!;
  expect(row).toHaveTextContent('19');
  expect(screen.getByText('Cost per part number')).toBeInTheDocument();
});

it('reloads with the chosen period and carries it into the exports', async () => {
  const fetcher = vi.fn().mockResolvedValue(respond(dashboard));
  vi.stubGlobal('fetch', fetcher);
  render(<ProductionDashboardView />);
  await screen.findByText('Output per part');
  fireEvent.change(screen.getByLabelText('From date'), { target: { value: '2026-08-01' } });
  fireEvent.change(screen.getByLabelText('To date'), { target: { value: '2026-08-31' } });
  fireEvent.click(screen.getByRole('button', { name: 'Apply period' }));
  await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(2));
  expect(fetcher.mock.calls[1][0]).toBe('/api/production-dashboard?from=2026-08-01&to=2026-08-31');
  expect(screen.getByRole('link', { name: 'Export Excel' })).toHaveAttribute(
    'href',
    '/api/production-reports/dashboard.xlsx?from=2026-09-01&to=2026-09-15',
  );
});

it('hides cost from the floor but keeps the quantity view', async () => {
  auth.permissions = ['production.view'];
  const withoutCosts = { ...dashboard, costs: [], totals: { ...dashboard.totals, totalActual: '0', wipValue: '0' } };
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond(withoutCosts)));
  render(<ProductionDashboardView />);
  await screen.findByText('Output per part');
  expect(screen.queryByText('Cost per part number')).not.toBeInTheDocument();
  expect(screen.queryByRole('link', { name: 'Export Excel' })).not.toBeInTheDocument();
  expect(screen.getByLabelText('STAMPING: 3 in progress')).toBeInTheDocument();
});

it('reports money in the single reporting currency', async () => {
  const format = (value: number) => new Intl.NumberFormat(undefined, { style: 'currency', currency: 'IDR', maximumFractionDigits: 2 }).format(value).replace(/\s/g, ' ');
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond(dashboard)));
  render(<ProductionDashboardView />);
  expect((await screen.findAllByText(format(610))).length).toBeGreaterThan(0);
  expect(screen.getAllByText(format(150)).length).toBeGreaterThan(0);
  expect(screen.getByLabelText('STAMPING: 3 in progress')).toBeInTheDocument();
});
