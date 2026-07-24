import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { localDateISO, SupplierOrderForm } from './supplier-order-form';
import { SupplierOrderIndex } from './supplier-order-index';
import { formatDecimal, multiplyDecimals } from './supplier-order-form';

const push = vi.fn();
vi.mock('next/navigation', () => ({ useRouter: () => ({ push, replace: push }) }));

const supplier = { id: 'supplier-1', code: 'SUP-01', name: 'PT Prima', currency: 'IDR', active: true };
const material = { id: 'material-1', code: 'RM-01', name: 'Steel Coil', supplierId: 'supplier-1', baseUnitCode: 'KG', qtyPerKanban: '0.1', standardUnitPrice: '0.2', active: true };

function response(body: unknown, ok = true) { return { ok, json: async () => body } as Response; }
function deferred<T>() { let resolve!: (value: T) => void; const promise = new Promise<T>(value => { resolve = value; }); return { promise, resolve }; }

describe('supplier order UI', () => {
  beforeEach(() => { push.mockReset(); vi.stubGlobal('confirm', vi.fn(() => true)); });

  it('renders real purchase orders and opens the dedicated create page', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({ items: [{ id: 'po-1', poNumber: 'PO-202607-00001', supplierId: 'supplier-1', supplierName: 'SUP-01 â€” PT Prima', orderDate: '2026-07-21T00:00:00Z', expectedDeliveryDate: '2026-07-25T00:00:00Z', status: 'DRAFT', totalAmount: '12.500000', currency: 'IDR', createdBy: { displayName: 'Admin' } }], total: 1 })));
    const user = userEvent.setup();
    render(<SupplierOrderIndex permissions={['po.view', 'po.price.view']} />);
    expect(await screen.findByText('PO-202607-00001')).toBeInTheDocument();
    expect(screen.getByText('SUP-01 â€” PT Prima')).toBeInTheDocument();
    expect(screen.getByText('IDR 12')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Actions for PO-202607-00001' })).toBeInTheDocument();
    expect(fetch).not.toHaveBeenCalledWith('/api/master-data/suppliers?limit=200', { credentials: 'include' });
    await user.click(screen.getByRole('button', { name: 'Create order' }));
    expect(push).toHaveBeenCalledWith('/supplier-orders/new');
  });

  it('does not render price columns for a Viewer', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({ items: [{ id: 'po-1', poNumber: 'PO-VIEWER', supplierId: 'supplier-1', supplierName: 'PT Prima', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-25', status: 'APPROVED', currency: 'IDR', createdBy: { displayName: 'Admin' } }], total: 1 })));

    render(<SupplierOrderIndex permissions={['po.view']} />);

    expect(await screen.findByText('PO-VIEWER')).toBeInTheDocument();
    expect(screen.queryByRole('columnheader', { name: 'Total' })).not.toBeInTheDocument();
    expect(screen.queryByText(/IDR/)).not.toBeInTheDocument();
  });

  it('shows compact new-tab PDF actions only when each document is available', async () => {
    const user = userEvent.setup();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({ items: [
      { id: 'po-approved', poNumber: 'PO-APPROVED', supplierId: 'supplier-1', supplierName: 'PT Prima', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-25', status: 'APPROVED', currency: 'IDR', documents: { deliveryNoteId: 'dn-1', deliveryNoteNumber: 'DN-202607-00001', kanbanCount: 1000, issuedAt: '2026-07-22T00:00:00Z' } },
      { id: 'po-approved-empty', poNumber: 'PO-APPROVED-EMPTY', supplierId: 'supplier-1', supplierName: 'PT Prima', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-25', status: 'APPROVED', currency: 'IDR', documents: null },
      { id: 'po-approved-oversized', poNumber: 'PO-APPROVED-OVERSIZED', supplierId: 'supplier-1', supplierName: 'PT Prima', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-25', status: 'APPROVED', currency: 'IDR', documents: { deliveryNoteId: 'dn-oversized', deliveryNoteNumber: 'DN-OVERSIZED', kanbanCount: 1001, issuedAt: '2026-07-22T00:00:00Z' } },
      { id: 'po-pending', poNumber: 'PO-PENDING', supplierId: 'supplier-1', supplierName: 'PT Prima', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-25', status: 'PENDING_APPROVAL', currency: 'IDR', documents: { deliveryNoteId: 'dn-pending', deliveryNoteNumber: 'DN-PENDING', kanbanCount: 2, issuedAt: '2026-07-22T00:00:00Z' } }
    ], total: 4 })));

    render(<SupplierOrderIndex permissions={['po.view']} />);

    const approvedRow = (await screen.findByText('PO-APPROVED')).closest('tr')!;
    const approvedEmptyRow = screen.getByText('PO-APPROVED-EMPTY').closest('tr')!;
    const oversizedRow = screen.getByText('PO-APPROVED-OVERSIZED').closest('tr')!;
    const pendingRow = screen.getByText('PO-PENDING').closest('tr')!;
    expect(screen.getByRole('columnheader', { name: 'No.' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Actions' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Documents' })).toBeInTheDocument();

    await user.click(within(approvedRow).getByRole('button', { name: 'Documents for PO-APPROVED' }));
    const poLink = within(approvedRow).getByRole('link', { name: 'Purchase Order PDF for PO-APPROVED' });
    expect(poLink).toHaveAttribute('href', '/api/purchase-orders/po-approved/documents/po.pdf');
    expect(poLink).toHaveAttribute('target', '_blank');
    expect(poLink).toHaveAttribute('rel', 'noopener noreferrer');
    const deliveryNoteLink = within(approvedRow).getByRole('link', { name: 'Delivery Note PDF for PO-APPROVED' });
    expect(deliveryNoteLink).toHaveAttribute('href', '/api/purchase-orders/po-approved/documents/delivery-note.pdf');
    expect(deliveryNoteLink).toHaveAttribute('target', '_blank');
    expect(deliveryNoteLink).toHaveAttribute('rel', 'noopener noreferrer');
    const labelsLink = within(approvedRow).getByRole('link', { name: 'Kanban Labels PDF for PO-APPROVED' });
    expect(labelsLink).toHaveAttribute('href', '/api/purchase-orders/po-approved/documents/kanban-labels.pdf');
    expect(labelsLink).toHaveAttribute('target', '_blank');
    expect(labelsLink).toHaveAttribute('rel', 'noopener noreferrer');

    await user.click(within(approvedEmptyRow).getByRole('button', { name: 'Documents for PO-APPROVED-EMPTY' }));
    expect(within(approvedEmptyRow).getByRole('link', { name: 'Purchase Order PDF for PO-APPROVED-EMPTY' })).toHaveAttribute('target', '_blank');
    expect(within(approvedEmptyRow).getByRole('button', { name: 'Delivery Note PDF' })).toBeDisabled();
    await user.click(within(oversizedRow).getByRole('button', { name: 'Documents for PO-APPROVED-OVERSIZED' }));
    expect(within(oversizedRow).getByRole('link', { name: 'Delivery Note PDF for PO-APPROVED-OVERSIZED' })).toBeInTheDocument();
    expect(within(oversizedRow).getByRole('button', { name: 'Kanban Labels PDF' })).toBeDisabled();
    await user.click(within(pendingRow).getByRole('button', { name: 'Documents for PO-PENDING' }));
    expect(within(pendingRow).getByRole('link', { name: 'Purchase Order PDF for PO-PENDING' })).toHaveAttribute('target', '_blank');
    expect(within(pendingRow).getByRole('button', { name: 'Delivery Note PDF' })).toBeDisabled();
  });

  it('cancels drafts from a compact row menu once and reloads the list', async () => {
    const orders = [
      { id: 'po-draft', poNumber: 'PO-DRAFT', supplierId: 'supplier-1', supplierName: 'PT Prima', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-25', status: 'DRAFT', currency: 'IDR', createdBy: { displayName: 'Admin' } },
      { id: 'po-approved', poNumber: 'PO-APPROVED', supplierId: 'supplier-1', supplierName: 'PT Prima', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-25', status: 'APPROVED', currency: 'IDR', createdBy: { displayName: 'Admin' } }
    ];
    const cancellation = deferred<Response>();
    let listLoads = 0;
    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url === '/api/purchase-orders/po-draft/cancel' && init?.method === 'POST') return cancellation.promise;
      const items = listLoads === 0 ? orders : [{ ...orders[0], status: 'CANCELLED' }, orders[1]];
      listLoads += 1;
      return Promise.resolve(response({ items, total: items.length }));
    });
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    render(<SupplierOrderIndex permissions={['po.view', 'po.edit_draft']} />);

    const actions = await screen.findByRole('button', { name: 'Actions for PO-DRAFT' });
    expect(screen.getByRole('button', { name: 'Actions for PO-APPROVED' })).toBeInTheDocument();
    await user.click(actions);
    expect(actions).not.toHaveAttribute('aria-haspopup');
    expect(screen.getByTestId('draft-actions-po-draft')).toHaveClass('row-menu-list');
    await user.click(screen.getByRole('button', { name: 'Cancel Draft' }));
    const dialog = screen.getByRole('dialog', { name: 'Cancel draft PO-DRAFT' });
    const confirm = screen.getByRole('button', { name: 'Confirm cancellation' });
    expect(confirm).toHaveFocus();
    await user.tab();
    expect(within(dialog).getByRole('button', { name: 'Close cancel draft confirmation' })).toHaveFocus();
    await user.tab({ shift: true });
    expect(confirm).toHaveFocus();

    await user.click(confirm);
    await waitFor(() => expect(dialog).toHaveFocus());
    await user.tab();
    expect(dialog).toHaveFocus();
    await user.click(confirm);
    expect(fetchMock.mock.calls.filter(([url, init]) => url === '/api/purchase-orders/po-draft/cancel' && (init as RequestInit | undefined)?.method === 'POST')).toHaveLength(1);

    cancellation.resolve(response({ ...orders[0], status: 'CANCELLED' }));
    await waitFor(() => expect(listLoads).toBe(2));
    expect(screen.queryByRole('dialog', { name: 'Cancel draft PO-DRAFT' })).not.toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole('button', { name: 'Actions for PO-DRAFT' })).toHaveFocus());
  });

  it('does not offer draft cancellation without draft-edit permission', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({ items: [{ id: 'po-draft', poNumber: 'PO-DRAFT', supplierId: 'supplier-1', supplierName: 'PT Prima', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-25', status: 'DRAFT', currency: 'IDR' }], total: 1 })));

    render(<SupplierOrderIndex permissions={['po.view']} />);

    const actions = await screen.findByRole('button', { name: 'Actions for PO-DRAFT' });
    await userEvent.click(actions);
    expect(screen.getByRole('button', { name: 'Cancel Draft' })).toBeDisabled();
  });

  it('shows cancellation errors and returns focus when Escape closes the confirmation', async () => {
    const order = { id: 'po-draft', poNumber: 'PO-DRAFT', supplierId: 'supplier-1', supplierName: 'PT Prima', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-25', status: 'DRAFT', currency: 'IDR', createdBy: { displayName: 'Admin' } };
    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => Promise.resolve(
      url === '/api/purchase-orders/po-draft/cancel' && init?.method === 'POST'
        ? response({ message: 'Draft can no longer be cancelled' }, false)
        : response({ items: [order], total: 1 })
    ));
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    render(<SupplierOrderIndex permissions={['po.view', 'po.edit_draft']} />);

    const actions = await screen.findByRole('button', { name: 'Actions for PO-DRAFT' });
    await user.click(actions);
    await user.click(screen.getByRole('button', { name: 'Cancel Draft' }));
    await user.click(screen.getByRole('button', { name: 'Confirm cancellation' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Draft can no longer be cancelled');
    expect(screen.getByRole('button', { name: 'Confirm cancellation' })).toBeEnabled();
    const dialog = screen.getByRole('dialog', { name: 'Cancel draft PO-DRAFT' });
    expect(dialog).toHaveFocus();
    await user.tab();
    expect(within(dialog).getByRole('button', { name: 'Close cancel draft confirmation' })).toHaveFocus();
    dialog.focus();
    await user.tab({ shift: true });
    expect(screen.getByRole('button', { name: 'Confirm cancellation' })).toHaveFocus();

    await user.keyboard('{Escape}');
    expect(screen.queryByRole('dialog', { name: 'Cancel draft PO-DRAFT' })).not.toBeInTheDocument();
    expect(actions).toHaveFocus();
  });

  it('ignores an out-of-order earlier index response after search changes', async () => {
    vi.useFakeTimers();
    try {
      const firstOrders = deferred<Response>();
      const secondOrders = deferred<Response>();
      let request = 0;
      const fetchMock = vi.fn().mockImplementation(() => [firstOrders, secondOrders][request++]?.promise);
      vi.stubGlobal('fetch', fetchMock);
      render(<SupplierOrderIndex permissions={['po.view']} />);
      await act(async () => { await vi.advanceTimersByTimeAsync(200); });
      fireEvent.change(screen.getByRole('searchbox', { name: 'Search Supplier Orders' }), { target: { value: 'new' } });
      await act(async () => { await vi.advanceTimersByTimeAsync(200); });
      secondOrders.resolve(response({ items: [{ id: 'new', poNumber: 'PO-NEW', supplierId: 'supplier-1', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', status: 'DRAFT', totalAmount: '1', currency: 'IDR' }], total: 1 }));
      await act(async () => {});
      expect(screen.getByText('PO-NEW')).toBeInTheDocument();
      firstOrders.resolve(response({ items: [{ id: 'old', poNumber: 'PO-OLD', supplierId: 'supplier-1', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', status: 'DRAFT', totalAmount: '1', currency: 'IDR' }], total: 1 }));
      await act(async () => {});
      expect(screen.queryByText('PO-OLD')).not.toBeInTheDocument();
      expect(screen.getByText('PO-NEW')).toBeInTheDocument();
    } finally { vi.useRealTimers(); }
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
    render(<SupplierOrderForm permissions={['po.price.view']} />);
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
    expect(screen.getAllByText('IDR 0').length).toBeGreaterThan(0);
    expect(screen.getByRole('combobox', { name: 'Raw Material' })).toHaveAttribute('list');
    await user.clear(screen.getByRole('spinbutton', { name: 'Total Kanban for Steel Coil' }));
    await user.type(screen.getByRole('spinbutton', { name: 'Total Kanban for Steel Coil' }), '2');
    expect(screen.getByText('0,2')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Save as Draft' }));
    await waitFor(() => expect(fetchMock).toHaveBeenLastCalledWith('/api/purchase-orders', expect.objectContaining({ method: 'POST' })));
    expect(JSON.parse((fetchMock.mock.calls[2][1] as RequestInit).body as string)).toEqual(expect.objectContaining({ supplierId: 'supplier-1', lines: [{ rawMaterialId: 'material-1', totalKanban: '2' }] }));
    await waitFor(() => expect(push).toHaveBeenCalledWith('/supplier-orders'));
    expect(screen.queryByRole('button', { name: 'Cancel draft' })).not.toBeInTheDocument();
    push.mockReset();
    await user.click(screen.getByRole('button', { name: 'Save & Send for Approval' }));
    await waitFor(() => expect(fetchMock).toHaveBeenLastCalledWith('/api/purchase-orders/po-1/submit', expect.objectContaining({ method: 'POST' })));
    expect(JSON.parse((fetchMock.mock.calls[3][1] as RequestInit).body as string)).toEqual(expect.objectContaining({ supplierId: 'supplier-1', lines: [{ rawMaterialId: 'material-1', totalKanban: '2' }] }));
    await waitFor(() => expect(push).toHaveBeenCalledWith('/supplier-orders'));
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

  it('keeps committed supplier lines while users incrementally type an alternative and restores on declined confirmation', async () => {
    const other = { ...supplier, id: 'supplier-2', code: 'SUP-02', name: 'PT Dua' };
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => Promise.resolve(response(url.includes('suppliers') ? { items: [supplier, other] } : { items: [material] }))));
    const confirmMock = vi.fn(() => false);
    vi.stubGlobal('confirm', confirmMock);
    const user = userEvent.setup();
    render(<SupplierOrderForm initialOrder={{ id: 'po-1', status: 'DRAFT', supplierId: 'supplier-1', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', lines: [{ rawMaterialId: 'material-1', rawMaterialCode: 'RM-01', rawMaterialName: 'Steel Coil', baseUnitCode: 'KG', qtyPerKanbanSnapshot: '1', totalKanban: '1', unitPriceSnapshot: '1' }] }} />);
    const supplierBox = screen.getByRole('combobox', { name: 'Supplier' });
    await waitFor(() => expect(supplierBox).toHaveValue('SUP-01 — PT Prima'));
    await user.clear(supplierBox);
    await user.type(supplierBox, 'SUP-02 — PT Dua');
    expect(confirmMock).toHaveBeenCalledTimes(1);
    expect(screen.getByText('RM-01 — Steel Coil')).toBeInTheDocument();
    expect(supplierBox).toHaveValue('SUP-01 — PT Prima');
  });

  it('shows save errors without offering draft cancellation inside the form', async () => {
    const fetchMock = vi.fn().mockResolvedValue(response({ message: 'Cannot save this order', fields: { expectedDeliveryDate: 'Expected date is invalid' } }, false));
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    render(<SupplierOrderForm initialOrder={{ id: 'po-1', status: 'DRAFT', supplierId: 'supplier-1', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', lines: [] }} />);
    await user.click(screen.getByRole('button', { name: 'Save as Draft' }));
    expect(await screen.findByText('Cannot save this order')).toBeInTheDocument();
    expect(screen.getByText('Expected date is invalid')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Cancel draft' })).not.toBeInTheDocument();
  });

  it('rounds decimal display and calculations half away from zero at six decimal places', () => {
    expect(formatDecimal('1.0000005')).toBe('1.000001');
    expect(formatDecimal('-1.0000005')).toBe('-1.000001');
    expect(multiplyDecimals('0.3333335', '3')).toBe('1.000001');
  });

  it('derives default order dates from local calendar components', () => {
    expect(localDateISO(new Date(2026, 0, 1, 0, 5))).toBe('2026-01-01');
    expect(localDateISO(new Date(2026, 11, 31, 23, 55))).toBe('2026-12-31');
  });

  it('clears stale material selection while restoring committed supplier after an unmatched query blur', async () => {
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
    await user.tab();
    expect(supplierBox).toHaveValue('SUP-01 — PT Prima');
    expect(screen.getByRole('combobox', { name: 'Raw Material' })).toBeEnabled();
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
    expect(push).not.toHaveBeenCalled();
  });

  it('promotes approver errors and associates raw material errors with their row', async () => {
    const order = { id: 'po-1', poNumber: 'PO-1', status: 'DRAFT', supplierId: 'supplier-1', supplierName: 'PT Prima', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', lines: [{ rawMaterialId: 'material-1', rawMaterialCode: 'RM-01', rawMaterialName: 'Steel Coil', baseUnitCode: 'KG', qtyPerKanbanSnapshot: '1', totalKanban: '1', unitPriceSnapshot: '2' }] };
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.includes('/submit')) return Promise.resolve(response({ message: 'Purchase order conflicts with its current state', fields: { approver: 'Configure an active PO Approver before submission', 'lines[0].rawMaterialId': 'Raw Material is no longer active for this supplier' } }, false));
      if (init?.method === 'PUT') return Promise.resolve(response(order));
      if (url.includes('/suppliers')) return Promise.resolve(response({ items: [supplier] }));
      return Promise.resolve(response({ items: [material] }));
    }));
    const user = userEvent.setup();
    render(<SupplierOrderForm initialOrder={order} permissions={['po.view', 'po.price.view']} />);

    await user.click(screen.getByRole('button', { name: 'Save & Send for Approval' }));

    expect(await screen.findByText('Configure an active PO Approver before submission')).toBeInTheDocument();
    expect(screen.getByText('Raw Material is no longer active for this supplier')).toBeInTheDocument();
    expect(screen.getByText('Raw Material is no longer active for this supplier').closest('td')).toHaveAttribute('aria-describedby', 'material-error-0');
    expect(push).not.toHaveBeenCalled();
  });

  it('loads a submitted Director detail from snapshots without editable master lists', async () => {
    const fetchMock = vi.fn().mockResolvedValue(response({ id: 'po-director', poNumber: 'PO-DIRECTOR', status: 'PENDING_APPROVAL', supplierId: 'supplier-1', supplierName: 'PT Prima snapshot', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', lines: [{ rawMaterialId: 'material-1', rawMaterialCode: 'RM-SNAPSHOT', rawMaterialName: 'Stored Steel', baseUnitCode: 'KG', qtyPerKanbanSnapshot: '3', totalKanban: '2', unitPriceSnapshot: '7' }] }));
    vi.stubGlobal('fetch', fetchMock);

    render(<SupplierOrderForm orderId="po-director" permissions={['po.view', 'po.price.view', 'po.approve', 'po.reject', 'inventory.view']} />);

    expect(await screen.findByText('PO-DIRECTOR')).toBeInTheDocument();
    expect(screen.getByText('PT Prima snapshot')).toBeInTheDocument();
    expect(screen.getByText(/RM-SNAPSHOT/)).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Unit Price' })).toBeInTheDocument();
    expect(screen.getByText('IDR 7')).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock).toHaveBeenCalledWith('/api/purchase-orders/po-director', { credentials: 'include' });
    expect(fetchMock).toHaveBeenCalledWith('/api/purchase-order-approvals?limit=200&offset=0', { credentials: 'include' });
    expect(screen.queryByText(/could not be loaded/i)).not.toBeInTheDocument();
  });

  it('lets the assigned Director approve a pending order from its detail page', async () => {
    const pendingOrder = { id: 'po-director', poNumber: 'PO-DIRECTOR', status: 'PENDING_APPROVAL', supplierId: 'supplier-1', supplierName: 'PT Prima', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', lines: [] };
    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url === '/api/purchase-order-approvals?limit=200&offset=0') return Promise.resolve(response({ items: [{ id: 'approval-1', purchaseOrderId: 'po-director', poNumber: 'PO-DIRECTOR', status: 'PENDING' }] }));
      if (url === '/api/purchase-order-approvals/approval-1/approve' && init?.method === 'POST') return Promise.resolve(response({ status: 'APPROVED' }));
      if (url === '/api/purchase-orders/po-director') return Promise.resolve(response({ ...pendingOrder, status: 'APPROVED' }));
      return Promise.resolve(response({ items: [] }));
    });
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    render(<SupplierOrderForm initialOrder={pendingOrder} permissions={['po.view', 'po.approve', 'po.reject']} />);

    await user.click(await screen.findByRole('button', { name: 'Approve PO-DIRECTOR' }));
    await user.click(screen.getByRole('button', { name: 'Approve order' }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/purchase-order-approvals/approval-1/approve', expect.objectContaining({ method: 'POST', body: '{}' })));
    expect(await screen.findByText('Approved')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Approve PO-DIRECTOR' })).not.toBeInTheDocument();
  });

  it('requires a reason when the assigned Director rejects from the order detail page', async () => {
    const pendingOrder = { id: 'po-director', poNumber: 'PO-DIRECTOR', status: 'PENDING_APPROVAL', supplierId: 'supplier-1', supplierName: 'PT Prima', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', lines: [] };
    const fetchMock = vi.fn().mockResolvedValue(response({ items: [{ id: 'approval-1', purchaseOrderId: 'po-director', poNumber: 'PO-DIRECTOR', status: 'PENDING' }] }));
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    render(<SupplierOrderForm initialOrder={pendingOrder} permissions={['po.view', 'po.approve', 'po.reject']} />);

    await user.click(await screen.findByRole('button', { name: 'Reject PO-DIRECTOR' }));
    await user.click(screen.getByRole('button', { name: 'Reject order' }));

    expect(await screen.findByText('Rejection reason is required')).toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalledWith('/api/purchase-order-approvals/approval-1/reject', expect.anything());
  });

  it('hides price and amount values in Viewer detail instead of rendering zeroes', () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({ items: [] })));
    render(<SupplierOrderForm initialOrder={{ id: 'po-viewer', poNumber: 'PO-VIEWER', status: 'APPROVED', supplierId: 'supplier-1', supplierName: 'PT Prima', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', lines: [{ rawMaterialId: 'material-1', rawMaterialCode: 'RM-01', rawMaterialName: 'Steel', baseUnitCode: 'KG', qtyPerKanbanSnapshot: '1', totalKanban: '2', unitPriceSnapshot: '99' }] }} permissions={['po.view']} />);

    expect(screen.queryByRole('columnheader', { name: 'Unit Price' })).not.toBeInTheDocument();
    expect(screen.queryByRole('columnheader', { name: 'Amount' })).not.toBeInTheDocument();
    expect(screen.queryByText('99.000000')).not.toBeInTheDocument();
    expect(screen.queryByText('Order Total')).not.toBeInTheDocument();
  });

  it('provides detail navigation without a redundant document card', async () => {
    const user = userEvent.setup();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({ items: [] })));
    const approvedOrder = { id: 'po-approved', poNumber: 'PO-APPROVED', status: 'APPROVED', supplierId: 'supplier-1', supplierName: 'PT Prima', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', lines: [], documents: { deliveryNoteId: 'dn-1', deliveryNoteNumber: 'DN-202607-00001', kanbanCount: 10, issuedAt: '2026-07-22T00:00:00Z' } };
    const { rerender } = render(<SupplierOrderForm initialOrder={approvedOrder} />);

    expect(screen.getByRole('button', { name: 'Back to Supplier Orders' })).toBeInTheDocument();
    expect(screen.queryByText('Documents')).not.toBeInTheDocument();
    expect(screen.queryByText('DN-202607-00001')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Print DN' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Print Kanban Labels' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Back to Supplier Orders' }));
    expect(push).toHaveBeenCalledWith('/supplier-orders');

    rerender(<SupplierOrderForm key="pending-order" initialOrder={{ ...approvedOrder, id: 'po-pending', status: 'PENDING_APPROVAL' }} />);
    expect(screen.queryByText('Documents')).not.toBeInTheDocument();
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
    expect(screen.getByRole('button', { name: 'Back to Supplier Orders' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Save as Draft' })).not.toBeInTheDocument();
    const retry = await screen.findByRole('button', { name: 'Retry' });
    expect(screen.getByRole('button', { name: 'Back to Supplier Orders' })).toBeInTheDocument();
    await user.click(retry);
    expect(await screen.findByText('PO-1')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Back to Supplier Orders' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Save as Draft' })).not.toBeInTheDocument();
    expect(screen.getByRole('status', { name: 'Supplier' })).toHaveTextContent('supplier-1');
  });

  it('surfaces failed supplier and material lookups with the detail form intact', async () => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => Promise.resolve(response(url.includes('suppliers') ? {} : {}, false))));
    render(<SupplierOrderForm initialOrder={{ id: 'po-1', status: 'DRAFT', supplierId: 'supplier-1', currency: 'IDR', orderDate: '2026-07-21', expectedDeliveryDate: '2026-07-22', lines: [] }} />);
    expect(await screen.findByText('Suppliers could not be loaded.')).toBeInTheDocument();
    expect(await screen.findByText('Raw Materials could not be loaded.')).toBeInTheDocument();
  });
});
