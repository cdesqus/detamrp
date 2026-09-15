import { render, screen, fireEvent } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { CostIndex, CostDetail } from './cost-pages';

vi.mock('next/navigation', () => ({ useRouter: () => ({ push: vi.fn() }) }));
vi.mock('../app-shell/app-shell', () => ({ useCurrentUser: () => ({ permissions: ['production.view', 'production.report'] }) }));

const cost = {
  orderId: 'o1',
  orderNumber: 'PRO-0000001',
  planNumber: 'PP-202609-000001',
  partNumber: 'FG-001',
  partName: 'Panel',
  unitCode: 'PCS',
  plantName: 'Main plant',
  status: 'IN_PROGRESS',
  currency: 'IDR',
  periodStart: '2026-09-01',
  dueDate: '2026-09-30',
  plannedQty: '100',
  goodQty: '19',
  rejectQty: '3',
  materialEstimate: '1000',
  processEstimate: '1300',
  totalEstimate: '2300',
  materialActual: '300',
  processActual: '310',
  totalActual: '610',
  wipValue: '150',
  finishedCost: '460',
  plannedUnitCost: '23',
  actualUnitCost: '24.210526',
  variance: '1.210526',
  variancePercent: '5.26',
  entryCount: 3,
  updatedAt: '2026-09-14T00:00:00Z',
  operations: [
    { operationId: 'op1', code: 'STAMPING', name: 'Stamping', sequence: 1, final: false, processedQty: '30', goodQty: '28', rejectQty: '2', rateSnapshot: '5', estimateCost: '500', actualCost: '150', costPerPiece: '5.357142', entries: 2 },
    { operationId: 'op2', code: 'WELDING', name: 'Welding', sequence: 2, final: true, processedQty: '20', goodQty: '19', rejectQty: '1', rateSnapshot: '8', estimateCost: '800', actualCost: '160', costPerPiece: '8.421052', entries: 1 },
  ],
  materials: [
    { materialId: 'm1', partNumber: 'RM-001', partName: 'Steel', unitCode: 'PCS', usedQty: '32', standardQty: '30', qtyVariance: '2', averagePrice: '10', actualCost: '320' },
  ],
};
const periods = [
  { period: '2026-09', closed: true, closedBy: 'Controller', orders: 1, entries: 3, processedQty: '50', goodQty: '47', rejectQty: '3', materialActual: '300', processActual: '310', totalActual: '610', currency: 'IDR' },
  { period: '2026-08', closed: false, closedBy: '', orders: 2, entries: 5, processedQty: '80', goodQty: '75', rejectQty: '5', materialActual: '500', processActual: '400', totalActual: '900', currency: 'IDR' },
];
const respond = (value: unknown) => ({ ok: true, json: async () => value });

beforeEach(() => vi.clearAllMocks());

function stubCostFetch() {
  return vi.fn((input: RequestInfo | URL) =>
    Promise.resolve(
      String(input).includes('/periods') ? respond({ items: periods }) : respond({ items: [cost] }),
    ),
  );
}

it('lists actual cost per order with its variance against the estimate', async () => {
  vi.stubGlobal('fetch', stubCostFetch());
  render(<CostIndex />);
  expect(await screen.findByText('PRO-0000001')).toBeInTheDocument();
  expect(screen.getByText('+5.26%')).toBeInTheDocument();
});

it('switches to the period summary and marks closed months', async () => {
  vi.stubGlobal('fetch', stubCostFetch());
  render(<CostIndex />);
  await screen.findByText('PRO-0000001');
  fireEvent.click(screen.getByRole('tab', { name: 'Per period' }));
  expect(screen.getByText('2026-09')).toBeInTheDocument();
  expect(screen.getByText('CLOSED')).toBeInTheDocument();
  expect(screen.getByText('OPEN')).toBeInTheDocument();
});

it('filters orders by production period', async () => {
  vi.stubGlobal('fetch', stubCostFetch());
  render(<CostIndex />);
  await screen.findByText('PRO-0000001');
  fireEvent.change(screen.getByLabelText('Period filter'), { target: { value: '2026-08' } });
  expect(screen.queryByText('PRO-0000001')).not.toBeInTheDocument();
});

it('breaks the order cost down into material, operations and WIP', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond(cost)));
  render(<CostDetail id="o1" />);
  expect(await screen.findByText('Cost build-up')).toBeInTheDocument();
  expect(screen.getByText('Less: cost held in WIP')).toBeInTheDocument();
  expect(screen.getAllByText('STAMPING').length).toBeGreaterThan(0);
  expect(screen.getAllByText('Finished output cost').length).toBeGreaterThan(1);
  expect(screen.getByText(/30 processed ×/)).toBeInTheDocument();
});

it('shows material usage against the BOM standard', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond(cost)));
  render(<CostDetail id="o1" />);
  expect(await screen.findByText('RM-001')).toBeInTheDocument();
  expect(screen.getByText('+2')).toBeInTheDocument();
});

it('lists each period once in the summary and in the filter', async () => {
  vi.stubGlobal('fetch', stubCostFetch());
  render(<CostIndex />);
  await screen.findByText('PRO-0000001');
  expect(screen.getAllByRole('option', { name: '2026-09' })).toHaveLength(1);
  fireEvent.click(screen.getByRole('tab', { name: 'Per period' }));
  expect(screen.getAllByText('2026-09').length).toBeLessThanOrEqual(2);
  expect(screen.getAllByText('IDR')).toHaveLength(periods.length);
});
