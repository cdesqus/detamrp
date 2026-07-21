import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SupplierOrderForm } from './supplier-order-form';
import { SupplierOrderIndex } from './supplier-order-index';

const push = vi.fn();
vi.mock('next/navigation', () => ({ useRouter: () => ({ push, replace: push }) }));

const supplier = { id: 'supplier-1', code: 'SUP-01', name: 'PT Prima', currency: 'IDR', active: true };
const material = { id: 'material-1', code: 'RM-01', name: 'Steel Coil', supplierId: 'supplier-1', baseUnitCode: 'KG', qtyPerKanban: '0.1', standardUnitPrice: '0.2', active: true };

function response(body: unknown, ok = true) { return { ok, json: async () => body } as Response; }

describe('supplier order UI', () => {
  beforeEach(() => { push.mockReset(); vi.stubGlobal('confirm', vi.fn(() => true)); });

  it('renders real purchase orders and opens the dedicated create page', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({ items: [{ id: 'po-1', poNumber: 'PO-202607-00001', supplierId: 'supplier-1', orderDate: '2026-07-21T00:00:00Z', expectedDeliveryDate: '2026-07-25T00:00:00Z', status: 'DRAFT', totalAmount: '12.500000', currency: 'IDR', createdBy: { displayName: 'Admin' } }], total: 1 })));
    const user = userEvent.setup();
    render(<SupplierOrderIndex />);
    expect(await screen.findByText('PO-202607-00001')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Create order' }));
    expect(push).toHaveBeenCalledWith('/supplier-orders/new');
  });

  it('filters material choices by searchable supplier, calculates without float drift, and sends both save actions', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(response({ items: [supplier], total: 1 }))
      .mockResolvedValueOnce(response({ items: [material], total: 1 }))
      .mockResolvedValueOnce(response({ id: 'po-1', poNumber: 'PO-202607-00001', status: 'DRAFT', lines: [] }))
      .mockResolvedValueOnce(response({ id: 'po-1', status: 'PENDING_APPROVAL' }));
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    render(<SupplierOrderForm />);
    const add = screen.getByRole('button', { name: '+ Raw Material' });
    expect(add).toBeDisabled();
    await user.type(screen.getByRole('combobox', { name: 'Supplier' }), 'PT Prima');
    expect(document.querySelector('#supplier-options option')).toHaveValue('SUP-01 — PT Prima');
    await user.clear(screen.getByRole('combobox', { name: 'Supplier' }));
    await user.type(screen.getByRole('combobox', { name: 'Supplier' }), 'SUP-01 — PT Prima');
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/master-data/raw-materials?active=true&limit=200&supplierId=supplier-1', expect.anything()));
    await waitFor(() => expect(document.querySelector('#material-options option')).toBeTruthy());
    await user.type(screen.getByRole('combobox', { name: 'Raw Material' }), 'Steel');
    expect(document.querySelector('#material-options option')).toHaveValue('RM-01 — Steel Coil');
    await user.clear(screen.getByRole('combobox', { name: 'Raw Material' }));
    await user.type(screen.getByRole('combobox', { name: 'Raw Material' }), 'RM-01 — Steel Coil');
    await user.click(add);
    expect(screen.getAllByText('0.020000').length).toBeGreaterThan(0);
    expect(screen.getByRole('combobox', { name: 'Raw Material' })).toHaveAttribute('list');
    await user.clear(screen.getByRole('spinbutton', { name: 'Total Kanban for Steel Coil' }));
    await user.type(screen.getByRole('spinbutton', { name: 'Total Kanban for Steel Coil' }), '2');
    expect(screen.getAllByText('0.040000').length).toBeGreaterThan(0);
    await user.click(screen.getByRole('button', { name: 'Save as Draft' }));
    await waitFor(() => expect(fetchMock).toHaveBeenLastCalledWith('/api/purchase-orders', expect.objectContaining({ method: 'POST' })));
    await user.click(screen.getByRole('button', { name: 'Save & Send for Approval' }));
    await waitFor(() => expect(fetchMock).toHaveBeenLastCalledWith('/api/purchase-orders/po-1/submit', expect.objectContaining({ method: 'POST' })));
  });

  it('requires confirmation before replacing a supplier with selected materials and prevents duplicate materials', async () => {
    const other = { ...supplier, id: 'supplier-2', code: 'SUP-02', name: 'PT Dua' };
    vi.stubGlobal('fetch', vi.fn()
      .mockResolvedValueOnce(response({ items: [supplier, other], total: 2 }))
      .mockResolvedValueOnce(response({ items: [material], total: 1 })));
    const user = userEvent.setup();
    const confirmMock = vi.fn(() => false);
    vi.stubGlobal('confirm', confirmMock);
    render(<SupplierOrderForm />);
    await user.type(screen.getByRole('combobox', { name: 'Supplier' }), 'PT Prima');
    await user.clear(screen.getByRole('combobox', { name: 'Supplier' }));
    await user.type(screen.getByRole('combobox', { name: 'Supplier' }), 'SUP-01 — PT Prima');
    await waitFor(() => expect(screen.getByRole('combobox', { name: 'Raw Material' })).toBeEnabled());
    await user.type(screen.getByRole('combobox', { name: 'Raw Material' }), 'Steel');
    await user.clear(screen.getByRole('combobox', { name: 'Raw Material' }));
    await user.type(screen.getByRole('combobox', { name: 'Raw Material' }), 'RM-01 — Steel Coil');
    await user.click(screen.getByRole('button', { name: '+ Raw Material' }));
    expect(document.querySelector('#material-options option')).toBeNull();
    await user.clear(screen.getByRole('combobox', { name: 'Supplier' }));
    await user.type(screen.getByRole('combobox', { name: 'Supplier' }), 'PT Dua');
    await user.clear(screen.getByRole('combobox', { name: 'Supplier' }));
    await user.type(screen.getByRole('combobox', { name: 'Supplier' }), 'SUP-02 — PT Dua');
    expect(confirmMock).toHaveBeenCalled();
    expect(screen.getByText('RM-01 — Steel Coil')).toBeInTheDocument();
  });

  it('shows server errors, lets a draft be cancelled, and keeps submitted details read-only', async () => {
    const fetchMock = vi.fn().mockResolvedValue(response({ message: 'Cannot save this order', fields: { expectedDeliveryDate: 'Expected date is invalid' } }, false));
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    render(<SupplierOrderForm initialOrder={{ id: 'po-1', status: 'DRAFT', supplierId: 'supplier-1', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', lines: [] }} />);
    await user.click(screen.getByRole('button', { name: 'Save as Draft' }));
    expect(await screen.findByText('Cannot save this order')).toBeInTheDocument();
    expect(screen.getByText('Expected date is invalid')).toBeInTheDocument();
    fetchMock.mockResolvedValueOnce(response({ id: 'po-1', status: 'CANCELLED' }));
    await user.click(screen.getByRole('button', { name: 'Cancel draft' }));
    await waitFor(() => expect(fetchMock).toHaveBeenLastCalledWith('/api/purchase-orders/po-1/cancel', expect.objectContaining({ method: 'POST' })));
  });
});
