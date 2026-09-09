import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SalesOrderForm } from './sales-order-form';

const push = vi.fn();
vi.mock('next/navigation', () => ({ useRouter: () => ({ push }) }));

describe('SalesOrderForm', () => {
  beforeEach(() => {
    push.mockReset();
    vi.stubGlobal('fetch', vi.fn(async (input, init) => {
      const url = String(input);
      if (url.includes('/customers')) return response({ items: [{ id: 'customer-1', code: 'CUS-1', name: 'Customer One' }] });
      if (url.includes('/finished-goods')) return response({ items: [{ id: 'fg-1', itemCode: 'FG-1', name: 'Finished One', baseUnitCode: 'PCS', salesPrice: '125000', currency: 'IDR' }] });
      if (url === '/api/sales-orders' && init?.method === 'POST') return response({ id: 'so-1' });
      throw new Error(`unexpected ${url}`);
    }));
  });

  it('shows master price as read-only and sends only customer and FG quantity', async () => {
    const user = userEvent.setup();
    render(<SalesOrderForm />);
    await screen.findByRole('option', { name: /Customer One/ });
    await user.selectOptions(screen.getByLabelText('Customer *'), 'customer-1');
    await user.selectOptions(screen.getByLabelText('Finished Good *'), 'fg-1');
    expect(screen.getByDisplayValue('125000')).toBeDisabled();
    await user.clear(screen.getByLabelText('Quantity *'));
    await user.type(screen.getByLabelText('Quantity *'), '2');
    await user.click(screen.getByRole('button', { name: 'Save draft' }));
    expect(fetch).toHaveBeenCalledWith('/api/sales-orders', expect.objectContaining({ body: expect.stringContaining('"customerId":"customer-1"') }));
    expect(push).toHaveBeenCalledWith('/sales-orders/so-1');
  });
});

function response(body: unknown) { return Promise.resolve({ ok: true, json: async () => body } as Response); }
