import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { InventoryIndex } from './inventory-index';

const response = {
  summary: {
    totalRawMaterials: 3,
    totalInStockKanban: 3,
    lowStockMaterials: 1,
    outOfStockMaterials: 1,
  },
  items: [
    {
      rawMaterialId: 'rm-zero',
      itemCode: 'RM-001',
      rawMaterialName: 'Bolt',
      supplierId: 'supplier-a',
      supplierName: 'Supplier A',
      availableKanban: 0,
      stockQuantity: '0.000000',
      baseUnitCode: 'PCS',
      minimumStock: '100.000000',
      stockStatus: 'OUT_OF_STOCK',
    },
    {
      rawMaterialId: 'rm-low',
      itemCode: 'RM-002',
      rawMaterialName: 'Belt',
      supplierId: 'supplier-a',
      supplierName: 'Supplier A',
      availableKanban: 1,
      stockQuantity: '50.000000',
      baseUnitCode: 'PCS',
      minimumStock: '50.000000',
      stockStatus: 'LOW_STOCK',
    },
    {
      rawMaterialId: 'rm-stock',
      itemCode: 'RM-003',
      rawMaterialName: 'Coil',
      supplierId: 'supplier-b',
      supplierName: 'Supplier B',
      availableKanban: 2,
      stockQuantity: '12.500000',
      baseUnitCode: 'KG',
      minimumStock: '5.000000',
      stockStatus: 'IN_STOCK',
    },
  ],
};

describe('InventoryIndex', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders summary and includes zero-stock materials', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => response,
    } as Response);

    render(<InventoryIndex />);

    expect(await screen.findByText('RM-001')).toBeInTheDocument();
    expect(screen.getAllByText('Out of Stock').find(element => element.classList.contains('status-pill'))).toHaveClass('status-pill--red');
    expect(screen.getAllByText('Low Stock').find(element => element.classList.contains('status-pill'))).toHaveClass('status-pill--amber');
    expect(screen.getAllByText('In Stock').find(element => element.classList.contains('status-pill'))).toHaveClass('status-pill--green');
    expect(screen.getByText('12,5')).toBeInTheDocument();
    expect(screen.getByText('Total Raw Materials').nextSibling).toHaveTextContent('3');
  });

  it('opens an in-stock Kanban detail modal', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce({ ok: true, json: async () => response } as Response)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          rawMaterialId: 'rm-stock',
          itemCode: 'RM-003',
          rawMaterialName: 'Coil',
          kanbans: [{
            kanbanLotId: 'lot-1',
            kanbanId: 'KB-001',
            deliveryNoteNumber: 'DN-001',
            poNumber: 'PO-001',
            quantity: '6.250000',
            baseUnitCode: 'KG',
            receivedDate: '2026-07-23T00:00:00Z',
          }],
        }),
      } as Response);

    render(<InventoryIndex />);
    await screen.findByText('RM-003');
    fireEvent.click(screen.getByRole('button', { name: 'Open RM-003' }));

    expect(await screen.findByRole('dialog', { name: 'In-stock Kanban — RM-003' })).toBeInTheDocument();
    expect(screen.getByText('KB-001')).toBeInTheDocument();
    expect(screen.getByText('6,25 KG')).toBeInTheDocument();
    expect(fetchMock).toHaveBeenLastCalledWith('/api/inventory/stock/rm-stock/kanbans', { credentials: 'include' });
  });

  it('sends supplier and status filters to the API', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => response,
    } as Response);
    render(<InventoryIndex />);
    await screen.findByText('RM-001');

    fireEvent.change(screen.getByLabelText('Supplier filter'), { target: { value: 'supplier-a' } });
    fireEvent.change(screen.getByLabelText('Stock status filter'), { target: { value: 'OUT_OF_STOCK' } });

    await waitFor(() => expect(fetchMock).toHaveBeenLastCalledWith(
      '/api/inventory/stock?supplierId=supplier-a&status=OUT_OF_STOCK',
      { credentials: 'include' },
    ));
  });
});
