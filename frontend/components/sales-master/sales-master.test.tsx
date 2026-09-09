import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Customers } from './customers';
import { FinishedGoods } from './finished-goods';

const finishedGood = {
  id: 'fg-1', itemCode: 'FG-100', name: 'Shared Assembly', baseUnitId: 'unit-1',
  baseUnitCode: 'PCS', baseUnitName: 'Pieces', salesPrice: '125000.500000', currency: 'IDR',
  priceVersion: 3, active: true,
  orderCustomers: ['Customer Alpha', 'Customer Beta']
};

describe('sales masters', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn(async input => {
      const url = String(input);
      if (url.includes('/master-data/units')) {
        return response({ items: [{ id: 'unit-1', code: 'PCS', name: 'Pieces', active: true }], total: 1 });
      }
      if (url.includes('/finished-goods')) return response({ items: [finishedGood], total: 1 });
      if (url.includes('/customers')) {
        return response({ items: [
          { id: 'customer-1', code: 'CUST-A', name: 'Customer Alpha', active: true },
          { id: 'customer-2', code: 'CUST-B', name: 'Customer Beta', active: true }
        ], total: 2 });
      }
      throw new Error(`unexpected URL ${url}`);
    }));
  });

  it('renders one shared finished good independently of two customer order contexts', async () => {
    render(<FinishedGoods permissions={['fg.view', 'fg.manage', 'fg.price.manage']} />);

    const rows = await screen.findAllByRole('row');
    expect(rows).toHaveLength(2);
    const dataRow = rows[1];
    expect(within(dataRow).getByText('FG-100')).toBeInTheDocument();
    expect(within(dataRow).getByText('Shared Assembly')).toBeInTheDocument();
    expect(within(dataRow).getByText('PCS')).toBeInTheDocument();
    expect(within(dataRow).getByText('125000.500000')).toBeInTheDocument();
    expect(screen.queryByRole('columnheader', { name: /customer/i })).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/customer/i)).not.toBeInTheDocument();
  });

  it('keeps metadata editing available but restricts finished-good price controls', async () => {
    const user = userEvent.setup();
    render(<FinishedGoods permissions={['fg.view', 'fg.manage']} />);

    expect(await screen.findByText('FG-100')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'New finished good' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Edit' }));

    expect(screen.getByRole('textbox', { name: /Finished Good Name/ })).toBeEnabled();
    expect(screen.getByRole('spinbutton', { name: /Sales Price/ })).toBeDisabled();
    expect(screen.getByRole('combobox', { name: /Currency/ })).toBeDisabled();
    expect(screen.getByText('Price changes require the FG Price Manage permission.')).toBeInTheDocument();
  });

  it('shows customer actions only to customer managers', async () => {
    render(<Customers permissions={['customer.view']} />);

    expect(await screen.findByText('Customer Alpha')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'New customer' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Edit' })).not.toBeInTheDocument();
  });
});

function response(body: unknown) {
  return Promise.resolve({ ok: true, json: async () => body } as Response);
}
