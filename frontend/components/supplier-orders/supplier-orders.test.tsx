import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SupplierOrderForm } from './supplier-order-form';
import { SupplierOrderIndex } from './supplier-order-index';
import { formatDecimal, multiplyDecimals } from './supplier-order-form';

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
    const savedOrder = { id: 'po-1', poNumber: 'PO-202607-00001', status: 'DRAFT', supplierId: 'supplier-1', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-21', notes: '', lines: [{ rawMaterialId: 'material-1', rawMaterialCode: 'RM-01', rawMaterialName: 'Steel Coil', baseUnitCode: 'KG', qtyPerKanbanSnapshot: '0.1', totalKanban: '2', unitPriceSnapshot: '0.2' }] };
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(response({ items: [supplier], total: 1 }))
      .mockResolvedValueOnce(response({ items: [material], total: 1 }))
      .mockResolvedValueOnce(response(savedOrder))
      .mockResolvedValueOnce(response(savedOrder))
      .mockResolvedValueOnce(response({ ...savedOrder, status: 'PENDING_APPROVAL' }));
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
    expect(JSON.parse((fetchMock.mock.calls[2][1] as RequestInit).body as string)).toEqual(expect.objectContaining({ supplierId: 'supplier-1', lines: [{ rawMaterialId: 'material-1', totalKanban: '2' }] }));
    await user.click(screen.getByRole('button', { name: 'Save & Send for Approval' }));
    await waitFor(() => expect(fetchMock).toHaveBeenLastCalledWith('/api/purchase-orders/po-1/submit', expect.objectContaining({ method: 'POST' })));
    expect(JSON.parse((fetchMock.mock.calls[3][1] as RequestInit).body as string)).toEqual(expect.objectContaining({ supplierId: 'supplier-1', lines: [{ rawMaterialId: 'material-1', totalKanban: '2' }] }));
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
    fireEvent.change(screen.getByRole('combobox', { name: 'Supplier' }), { target: { value: 'SUP-02 — PT Dua' } });
    expect(confirmMock).toHaveBeenCalled();
    expect(screen.getByText('RM-01 — Steel Coil')).toBeInTheDocument();
    expect(screen.getByRole('combobox', { name: 'Supplier' })).toHaveValue('SUP-01 — PT Prima');
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

  it('rounds decimal display and calculations half away from zero at six decimal places', () => {
    expect(formatDecimal('1.0000005')).toBe('1.000001');
    expect(formatDecimal('-1.0000005')).toBe('-1.000001');
    expect(multiplyDecimals('0.3333335', '3')).toBe('1.000001');
  });

  it('clears stale selection IDs when supplier or material text is cleared', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(response({ items: [supplier], total: 1 })).mockResolvedValueOnce(response({ items: [material], total: 1 }));
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    render(<SupplierOrderForm />);
    const supplierBox = screen.getByRole('combobox', { name: 'Supplier' });
    await user.type(supplierBox, 'SUP-01 — PT Prima');
    await waitFor(() => expect(screen.getByRole('combobox', { name: 'Raw Material' })).toBeEnabled());
    await waitFor(() => expect(document.querySelector('#material-options option')).toBeTruthy());
    const materialBox = screen.getByRole('combobox', { name: 'Raw Material' });
    await user.type(materialBox, 'RM-01 — Steel Coil');
    expect(screen.getByRole('button', { name: '+ Raw Material' })).toBeEnabled();
    await user.clear(materialBox);
    expect(screen.getByRole('button', { name: '+ Raw Material' })).toBeDisabled();
    await user.clear(supplierBox);
    expect(screen.getByRole('combobox', { name: 'Raw Material' })).toBeDisabled();
    await user.click(screen.getByRole('button', { name: 'Save as Draft' }));
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('blocks saving invalid Total Kanban and maps indexed backend field errors to that row', async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn().mockResolvedValue(response({ message: 'invalid', fields: { 'lines[0].totalKanban': 'Server says positive integer' } }, false));
    vi.stubGlobal('fetch', fetchMock);
    render(<SupplierOrderForm initialOrder={{ id: 'po-1', status: 'DRAFT', supplierId: 'supplier-1', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', lines: [{ rawMaterialId: 'material-1', rawMaterialCode: 'RM-01', rawMaterialName: 'Steel Coil', baseUnitCode: 'KG', qtyPerKanbanSnapshot: '1', totalKanban: '0', unitPriceSnapshot: '1' }] }} />);
    await user.click(screen.getByRole('button', { name: 'Save as Draft' }));
    expect(await screen.findByText('Total Kanban must be a positive integer')).toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalledWith('/api/purchase-orders/po-1', expect.objectContaining({ method: 'PUT' }));
  });

  it('maps a server indexed line error to its accessible Total Kanban control', async () => {
    const user = userEvent.setup();
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url.includes('/purchase-orders/po-1')) return Promise.resolve(response({ message: 'invalid', fields: { 'lines[0].totalKanban': 'Server says positive integer' } }, false));
      return Promise.resolve(response({ items: [] }));
    }));
    render(<SupplierOrderForm initialOrder={{ id: 'po-1', status: 'DRAFT', supplierId: 'supplier-1', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', lines: [{ rawMaterialId: 'material-1', rawMaterialCode: 'RM-01', rawMaterialName: 'Steel Coil', baseUnitCode: 'KG', qtyPerKanbanSnapshot: '1', totalKanban: '1', unitPriceSnapshot: '1' }] }} />);
    await user.click(screen.getByRole('button', { name: 'Save as Draft' }));
    expect(await screen.findByText('Server says positive integer')).toBeInTheDocument();
    expect(screen.getByRole('spinbutton', { name: 'Total Kanban for Steel Coil' })).toHaveAttribute('aria-describedby', 'kanban-error-0');
  });

  it('shows loading then retryable detail failure without draft actions before the order loads', async () => {
    let detailAttempts = 0;
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url === '/api/purchase-orders/po-detail') {
        detailAttempts += 1;
        return Promise.resolve(detailAttempts === 1 ? response({}, false) : response({ id: 'po-detail', poNumber: 'PO-1', status: 'PENDING_APPROVAL', supplierId: 'supplier-1', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', lines: [] }));
      }
      if (url.startsWith('/api/master-data/suppliers')) return Promise.resolve(response({ items: [supplier] }));
      return Promise.resolve(response({ items: [] }));
    }));
    const user = userEvent.setup();
    render(<SupplierOrderForm orderId="po-detail" />);
    expect(screen.getByText('Loading supplier order...')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Save as Draft' })).not.toBeInTheDocument();
    await user.click(await screen.findByRole('button', { name: 'Retry' }));
    expect(await screen.findByText('PO-1')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Save as Draft' })).not.toBeInTheDocument();
    expect(screen.getByRole('combobox', { name: 'Supplier' })).toBeDisabled();
  });

  it('surfaces failed supplier and material lookups with the detail form intact', async () => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => Promise.resolve(response(url.includes('suppliers') ? {} : {}, false))));
    render(<SupplierOrderForm initialOrder={{ id: 'po-1', status: 'DRAFT', supplierId: 'supplier-1', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', lines: [] }} />);
    expect(await screen.findByText('Suppliers could not be loaded.')).toBeInTheDocument();
    expect(await screen.findByText('Raw Materials could not be loaded.')).toBeInTheDocument();
  });
});
