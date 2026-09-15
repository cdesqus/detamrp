import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { RoutingIndex, RoutingForm, RoutingDetail } from './routing-pages';

const { push, auth } = vi.hoisted(() => ({ push: vi.fn(), auth: { permissions: ['production.view', 'production.routing', 'production.report'] } }));
vi.mock('next/navigation', () => ({ useRouter: () => ({ push }) }));
vi.mock('../app-shell/app-shell', () => ({ useCurrentUser: () => auth }));

const routing = {
  id: 'r1',
  partId: 'fg1',
  kind: 'FG',
  partNumber: 'FG-001',
  partName: 'Panel',
  unitCode: 'PCS',
  name: 'Panel routing',
  currency: 'IDR',
  revision: 2,
  active: true,
  steps: [
    { code: 'STAMPING', name: 'Stamping', rate: '250' },
    { code: 'WELDING', name: 'Welding', rate: '400' },
  ],
  totalRate: '650',
  linkedOrders: 0,
  updatedAt: '2026-09-14T00:00:00Z',
  createdBy: 'Planner',
  history: [{ action: 'UPDATE', actor: 'Planner', occurredAt: '2026-09-14T00:00:00Z' }],
};
const options = { parts: [{ id: 'fg1', kind: 'FG', partNumber: 'FG-001', partName: 'Panel', unitCode: 'PCS' }] };
const respond = (value: unknown) => ({ ok: true, json: async () => value });

beforeEach(() => {
  vi.clearAllMocks();
  auth.permissions = ['production.view', 'production.routing', 'production.report'];
});

it('lists routings with their operation flow and hides create from viewers', async () => {
  auth.permissions = ['production.view'];
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond({ items: [routing] })));
  render(<RoutingIndex />);
  expect(await screen.findByText('FG-001')).toBeInTheDocument();
  expect(screen.getByText('STAMPING → WELDING')).toBeInTheDocument();
  expect(screen.queryByText('New Routing')).not.toBeInTheDocument();
});

it('creates a routing with operations in sequence order', async () => {
  const fetcher = vi.fn().mockResolvedValueOnce(respond(options)).mockResolvedValueOnce(respond({ id: 'saved' }));
  vi.stubGlobal('fetch', fetcher);
  render(<RoutingForm />);
  await screen.findByRole('option', { name: 'FG-001 — Panel' });
  fireEvent.change(screen.getByLabelText('Part'), { target: { value: 'fg1' } });
  fireEvent.change(screen.getByLabelText('Routing name'), { target: { value: 'Panel routing' } });
  fireEvent.change(screen.getByLabelText('Operation code 1'), { target: { value: 'welding' } });
  fireEvent.change(screen.getByLabelText('Operation name 1'), { target: { value: 'Welding' } });
  fireEvent.change(screen.getByLabelText('Cost per piece 1'), { target: { value: '400' } });
  fireEvent.click(screen.getByRole('button', { name: '+ Add operation' }));
  fireEvent.change(screen.getByLabelText('Operation code 2'), { target: { value: 'STAMPING' } });
  fireEvent.change(screen.getByLabelText('Operation name 2'), { target: { value: 'Stamping' } });
  fireEvent.change(screen.getByLabelText('Cost per piece 2'), { target: { value: '250' } });
  fireEvent.click(screen.getByRole('button', { name: 'Move operation 2 up' }));
  fireEvent.click(screen.getByRole('button', { name: 'Create Routing' }));
  await waitFor(() => expect(push).toHaveBeenCalledWith('/production-routings/saved'));
  expect(JSON.parse(fetcher.mock.calls[1][1].body)).toEqual({
    partId: 'fg1',
    kind: 'FG',
    name: 'Panel routing',
    currency: 'IDR',
    steps: [
      { code: 'STAMPING', name: 'Stamping', rate: '250' },
      { code: 'WELDING', name: 'Welding', rate: '400' },
    ],
  });
});

it('rejects duplicate operation codes before calling the API', async () => {
  const fetcher = vi.fn().mockResolvedValue(respond(options));
  vi.stubGlobal('fetch', fetcher);
  render(<RoutingForm />);
  await screen.findByRole('option', { name: 'FG-001 — Panel' });
  fireEvent.change(screen.getByLabelText('Part'), { target: { value: 'fg1' } });
  fireEvent.change(screen.getByLabelText('Routing name'), { target: { value: 'Panel routing' } });
  fireEvent.change(screen.getByLabelText('Operation code 1'), { target: { value: 'STAMPING' } });
  fireEvent.change(screen.getByLabelText('Operation name 1'), { target: { value: 'Stamping' } });
  fireEvent.click(screen.getByRole('button', { name: '+ Add operation' }));
  fireEvent.change(screen.getByLabelText('Operation code 2'), { target: { value: 'stamping' } });
  fireEvent.change(screen.getByLabelText('Operation name 2'), { target: { value: 'Stamping again' } });
  fireEvent.submit(screen.getByRole('button', { name: 'Create Routing' }).closest('form')!);
  expect(await screen.findByRole('alert')).toHaveTextContent('unique');
  expect(fetcher).toHaveBeenCalledTimes(1);
});

it('freezes routings that production orders already reference', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond({ ...routing, linkedOrders: 3 })));
  render(<RoutingDetail id="r1" />);
  await screen.findByText('Panel routing · revision 2 · Panel');
  expect(screen.queryByRole('link', { name: 'Edit routing' })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Delete routing' })).not.toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'New revision' })).toHaveAttribute('href', '/production-routings/new?from=r1');
});

it('activates an inactive revision from the detail screen', async () => {
  const fetcher = vi
    .fn()
    .mockResolvedValueOnce(respond({ ...routing, active: false }))
    .mockResolvedValueOnce(respond({ ...routing, active: true }))
    .mockResolvedValue(respond({ ...routing, active: true }));
  vi.stubGlobal('fetch', fetcher);
  render(<RoutingDetail id="r1" />);
  fireEvent.click(await screen.findByRole('button', { name: 'Activate routing' }));
  await waitFor(() => expect(fetcher.mock.calls[1][0]).toBe('/api/production-routings/r1/activate'));
  expect(fetcher.mock.calls[1][1].method).toBe('POST');
  await waitFor(() => expect(screen.queryByRole('button', { name: 'Activate routing' })).not.toBeInTheDocument());
});

it('shows the input of each operation in the sequence', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respond(routing)));
  render(<RoutingDetail id="r1" />);
  expect(await screen.findByText('Raw material issue')).toBeInTheDocument();
  expect(screen.getByText('WIP STAMPING')).toBeInTheDocument();
});
