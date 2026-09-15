import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { WIPIndex, WIPDetail } from './wip-pages';

const { auth } = vi.hoisted(() => ({ auth: { permissions: ['production.view', 'production.wip', 'production.report'] } }));
vi.mock('next/navigation', () => ({ useRouter: () => ({ push: vi.fn() }) }));
vi.mock('../app-shell/app-shell', () => ({ useCurrentUser: () => auth }));

const NIL = '00000000-0000-0000-0000-000000000000';
const balance = {
  orderId: 'o1',
  orderNumber: 'PRO-0000001',
  planNumber: 'PP-202609-000001',
  partNumber: 'FG-001',
  partName: 'Panel',
  unitCode: 'PCS',
  plantName: 'Main plant',
  status: 'IN_PROGRESS',
  currency: 'IDR',
  plannedQty: '100',
  totalQty: '8',
  totalValue: '150',
  reconciled: true,
  updatedAt: '2026-09-14T00:00:00Z',
  operations: [
    {
      operationId: 'op1', code: 'STAMPING', name: 'Stamping', sequence: 1, final: false,
      goodQty: '28', processedQty: '30', received: '28', transferredOut: '25', transferredIn: '0', consumed: '0',
      onHand: '3', staged: '0', balance: '3', value: '56.25', reconciled: true, difference: '0',
    },
    {
      operationId: 'op2', code: 'WELDING', name: 'Welding', sequence: 2, final: true,
      goodQty: '19', processedQty: '20', received: '0', transferredOut: '0', transferredIn: '25', consumed: '20',
      onHand: '0', staged: '5', balance: '5', value: '93.75', reconciled: true, difference: '0',
    },
  ],
  movements: [
    {
      id: 'm1', orderId: 'o1', orderNumber: 'PRO-0000001', type: 'TRANSFER',
      sourceOperationId: 'op1', sourceCode: 'STAMPING', destinationOperationId: 'op2', destinationCode: 'WELDING',
      quantity: '25', unitCost: '15', totalCost: '375', currency: 'IDR', entryId: NIL, entryNumber: '',
      lotId: 'lot1', reversesId: NIL, reversed: false, movementDate: '2026-09-13', notes: 'To welding line',
      createdBy: 'Operator', createdAt: '2026-09-13T02:00:00Z', canReverse: false,
    },
    {
      id: 'm2', orderId: 'o1', orderNumber: 'PRO-0000001', type: 'RECEIPT',
      sourceOperationId: NIL, sourceCode: '', destinationOperationId: 'op1', destinationCode: 'STAMPING',
      quantity: '28', unitCost: '15', totalCost: '420', currency: 'IDR', entryId: 'e1', entryNumber: 'DP-0000001',
      lotId: NIL, reversesId: NIL, reversed: false, movementDate: '2026-09-10', notes: 'Good output of STAMPING',
      createdBy: 'Operator', createdAt: '2026-09-10T02:00:00Z', canReverse: false,
    },
  ],
};
const respond = (value: unknown) => ({ ok: true, json: async () => value });

beforeEach(() => {
  vi.clearAllMocks();
  auth.permissions = ['production.view', 'production.wip', 'production.report'];
});

it('lists WIP balances per operation', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond({ items: [balance] })));
  render(<WIPIndex />);
  expect(await screen.findByText('PRO-0000001')).toBeInTheDocument();
  expect(screen.getByText('STAMPING 3 · WELDING 5')).toBeInTheDocument();
  expect(screen.getByText('RECONCILED')).toBeInTheDocument();
});

it('hides WIP value from users without the cost permission', async () => {
  auth.permissions = ['production.view'];
  const withoutCosts = { ...balance, totalValue: undefined };
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond({ items: [withoutCosts] })));
  render(<WIPIndex />);
  await screen.findByText('PRO-0000001');
  expect(screen.queryByText('WIP value')).not.toBeInTheDocument();
});

it('transfers WIP to the next operation without exceeding the balance', async () => {
  const fetcher = vi.fn().mockResolvedValueOnce(respond(balance)).mockResolvedValueOnce(respond(balance));
  vi.stubGlobal('fetch', fetcher);
  render(<WIPDetail id="o1" />);
  fireEvent.click(await screen.findByRole('button', { name: 'Transfer WIP' }));
  fireEvent.change(screen.getByLabelText('Transfer quantity'), { target: { value: '9' } });
  fireEvent.click(screen.getByRole('button', { name: 'Post transfer' }));
  expect(await screen.findByRole('alert')).toHaveTextContent('exceeds the WIP available');
  expect(fetcher).toHaveBeenCalledTimes(1);
  fireEvent.change(screen.getByLabelText('Transfer quantity'), { target: { value: '2' } });
  fireEvent.change(screen.getByLabelText('Movement date'), { target: { value: '2026-09-15' } });
  fireEvent.click(screen.getByRole('button', { name: 'Post transfer' }));
  await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(2));
  expect(fetcher.mock.calls[1][0]).toBe('/api/production-wip/transfers');
  expect(JSON.parse(fetcher.mock.calls[1][1].body)).toMatchObject({
    orderId: 'o1',
    sourceOperationId: 'op1',
    quantity: '2',
    movementDate: '2026-09-15',
  });
});

it('shows the ledger and only offers reversal where the API allows it', async () => {
  const reversible = {
    ...balance,
    movements: [{ ...balance.movements[0], canReverse: true }, balance.movements[1]],
  };
  const fetcher = vi.fn().mockResolvedValueOnce(respond(reversible)).mockResolvedValueOnce(respond(balance));
  vi.stubGlobal('fetch', fetcher);
  render(<WIPDetail id="o1" />);
  expect(await screen.findByText('DP-0000001')).toBeInTheDocument();
  expect(screen.getAllByRole('button', { name: /Reverse transfer of/ })).toHaveLength(1);
  fireEvent.click(screen.getByRole('button', { name: /Reverse transfer of/ }));
  fireEvent.click(screen.getByRole('button', { name: 'Post reversal' }));
  await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(2));
  expect(fetcher.mock.calls[1][0]).toBe('/api/production-wip/movements/m1/reverse');
});

it('flags an operation whose ledger does not match its entries', async () => {
  const broken = {
    ...balance,
    reconciled: false,
    operations: [{ ...balance.operations[0], reconciled: false, difference: '-40' }, balance.operations[1]],
  };
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond(broken)));
  render(<WIPDetail id="o1" />);
  expect(await screen.findByText('CHECK LEDGER')).toBeInTheDocument();
  expect(screen.getByText('-40')).toBeInTheDocument();
});

it('totals WIP value in the reporting currency', async () => {
  const format = (value: number) => new Intl.NumberFormat(undefined, { style: 'currency', currency: 'IDR', maximumFractionDigits: 2 }).format(value).replace(/\s/g, ' ');
  const second = { ...balance, orderId: 'o2', orderNumber: 'PRO-0000002', totalQty: '4', totalValue: '25' };
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond({ items: [balance, second] })));
  render(<WIPIndex />);
  await screen.findByText('PRO-0000001');
  expect(screen.getByText(format(175))).toBeInTheDocument();
  expect(screen.getByText('12')).toBeInTheDocument();
});

it('offers the transfer straight from the balance list', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond({ items: [balance] })));
  render(<WIPIndex />);
  expect(await screen.findByRole('link', { name: 'Transfer WIP' })).toHaveAttribute('href', '/production-wip/o1');
});

it('explains the missing transfer button to users without the permission', async () => {
  auth.permissions = ['production.view'];
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond(balance)));
  render(<WIPDetail id="o1" />);
  await screen.findByText(/Transfer work in progress/);
  expect(screen.queryByRole('button', { name: 'Transfer WIP' })).not.toBeInTheDocument();
});
