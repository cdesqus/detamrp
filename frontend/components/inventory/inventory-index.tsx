'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { formatQuantity } from '../../lib/number-format';

type Summary = {
  totalRawMaterials: number;
  totalInStockKanban: number;
  lowStockMaterials: number;
  outOfStockMaterials: number;
};

type StockItem = {
  rawMaterialId: string;
  itemCode: string;
  rawMaterialName: string;
  supplierId: string;
  supplierName: string;
  availableKanban: number;
  stockQuantity: string;
  baseUnitCode: string;
  minimumStock: string;
  stockStatus: 'IN_STOCK' | 'LOW_STOCK' | 'OUT_OF_STOCK';
};

type KanbanItem = {
  kanbanLotId: string;
  kanbanId: string;
  deliveryNoteNumber: string;
  poNumber: string;
  quantity: string;
  baseUnitCode: string;
  receivedDate: string;
};

type KanbanResponse = {
  rawMaterialId: string;
  itemCode: string;
  rawMaterialName: string;
  kanbans: KanbanItem[];
};

const emptySummary: Summary = {
  totalRawMaterials: 0,
  totalInStockKanban: 0,
  lowStockMaterials: 0,
  outOfStockMaterials: 0,
};

const statusPresentation = {
  IN_STOCK: { label: 'In Stock', className: 'status-pill--green' },
  LOW_STOCK: { label: 'Low Stock', className: 'status-pill--amber' },
  OUT_OF_STOCK: { label: 'Out of Stock', className: 'status-pill--red' },
};

export function InventoryIndex() {
  const [summary, setSummary] = useState<Summary>(emptySummary);
  const [items, setItems] = useState<StockItem[]>([]);
  const [supplierOptions, setSupplierOptions] = useState<Array<{ id: string; name: string }>>([]);
  const [search, setSearch] = useState('');
  const [supplierId, setSupplierId] = useState('');
  const [status, setStatus] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [detail, setDetail] = useState<KanbanResponse | null>(null);
  const [detailTitle, setDetailTitle] = useState('');
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState('');

  const query = useMemo(() => {
    const params = new URLSearchParams();
    if (search.trim()) params.set('search', search.trim());
    if (supplierId) params.set('supplierId', supplierId);
    if (status) params.set('status', status);
    const value = params.toString();
    return value ? `?${value}` : '';
  }, [search, supplierId, status]);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const response = await fetch(`/api/inventory/stock${query}`, { credentials: 'include' });
      if (!response.ok) throw new Error('Stock inventory could not be loaded.');
      const payload = await response.json();
      const nextItems: StockItem[] = Array.isArray(payload.items) ? payload.items : [];
      setSummary(payload.summary ?? emptySummary);
      setItems(nextItems);
      if (!query) {
        const unique = new Map<string, string>();
        nextItems.forEach(item => unique.set(item.supplierId, item.supplierName));
        setSupplierOptions(Array.from(unique, ([id, name]) => ({ id, name })));
      }
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Stock inventory could not be loaded.');
    } finally {
      setLoading(false);
    }
  }, [query]);

  useEffect(() => {
    void load();
  }, [load]);

  async function openKanbans(item: StockItem) {
    setDetailTitle(item.itemCode);
    setDetail(null);
    setDetailError('');
    setDetailLoading(true);
    try {
      const response = await fetch(`/api/inventory/stock/${item.rawMaterialId}/kanbans`, { credentials: 'include' });
      if (!response.ok) throw new Error('In-stock Kanban could not be loaded.');
      const payload = await response.json();
      setDetail({ ...payload, kanbans: Array.isArray(payload.kanbans) ? payload.kanbans : [] });
    } catch (cause) {
      setDetailError(cause instanceof Error ? cause.message : 'In-stock Kanban could not be loaded.');
    } finally {
      setDetailLoading(false);
    }
  }

  function closeDetail() {
    setDetailTitle('');
    setDetail(null);
    setDetailError('');
  }

  return (
    <section className="module-index">
      <div className="page-title-row">
        <div>
          <h1>Stock Inventory</h1>
          <p className="muted">Real-time Raw Material stock based on received Kanban lots.</p>
        </div>
      </div>

      <div className="metric-grid inventory-metrics">
        <article><span>Total Raw Materials</span><strong>{summary.totalRawMaterials}</strong></article>
        <article><span>In-Stock Kanban</span><strong>{summary.totalInStockKanban}</strong></article>
        <article><span>Low Stock</span><strong>{summary.lowStockMaterials}</strong></article>
        <article><span>Out of Stock</span><strong>{summary.outOfStockMaterials}</strong></article>
      </div>

      <div className="table-frame inventory-table">
        <div className="table-toolbar">
          <input
            aria-label="Search stock"
            placeholder="Search item code or raw material"
            value={search}
            onChange={event => setSearch(event.target.value)}
          />
          <div className="inventory-filters">
            <select aria-label="Supplier filter" value={supplierId} onChange={event => setSupplierId(event.target.value)}>
              <option value="">All suppliers</option>
              {supplierOptions.map(option => <option key={option.id} value={option.id}>{option.name}</option>)}
            </select>
            <select aria-label="Stock status filter" value={status} onChange={event => setStatus(event.target.value)}>
              <option value="">All statuses</option>
              <option value="IN_STOCK">In Stock</option>
              <option value="LOW_STOCK">Low Stock</option>
              <option value="OUT_OF_STOCK">Out of Stock</option>
            </select>
          </div>
        </div>
        {error ? <div className="table-empty"><p className="form-error" role="alert">{error}</p><button className="table-action" onClick={() => void load()}>Retry</button></div> :
          <table>
            <thead><tr><th>Item Code</th><th>Raw Material</th><th>Supplier</th><th>Available Kanban</th><th>Stock Quantity</th><th>Base Unit</th><th>Minimum Stock</th><th>Status</th><th>Action</th></tr></thead>
            <tbody>
              {loading ? <tr><td colSpan={9}><div className="table-empty">Loading stock inventory...</div></td></tr> :
                items.length === 0 ? <tr><td colSpan={9}><div className="table-empty">No stock records found.</div></td></tr> :
                  items.map(item => {
                    const presentation = statusPresentation[item.stockStatus];
                    return <tr key={item.rawMaterialId}>
                      <td>{item.itemCode}</td>
                      <td>{item.rawMaterialName}</td>
                      <td>{item.supplierName}</td>
                      <td>{item.availableKanban}</td>
                      <td>{formatQuantity(item.stockQuantity)}</td>
                      <td>{item.baseUnitCode}</td>
                      <td>{formatQuantity(item.minimumStock)}</td>
                      <td><span className={`status-pill ${presentation.className}`}>{presentation.label}</span></td>
                      <td><button className="table-action" aria-label={`Open ${item.itemCode}`} onClick={() => void openKanbans(item)}>Open</button></td>
                    </tr>;
                  })}
            </tbody>
          </table>}
      </div>

      {detailTitle ? <>
        <button className="crud-scrim" aria-label="Close Kanban detail" onClick={closeDetail} />
        <div className="crud-modal crud-modal--wide inventory-detail-modal" role="dialog" aria-modal="true" aria-label={`In-stock Kanban — ${detailTitle}`}>
          <div className="crud-modal-heading">
            <div><strong>In-stock Kanban — {detailTitle}</strong><span>Kanban lots currently available for outgoing.</span></div>
            <button aria-label="Close Kanban detail" onClick={closeDetail}>×</button>
          </div>
          <div className="inventory-detail-body">
            {detailLoading ? <div className="table-empty">Loading in-stock Kanban...</div> :
              detailError ? <p className="form-error" role="alert">{detailError}</p> :
                <table>
                  <thead><tr><th>Kanban ID</th><th>DN Number</th><th>PO Number</th><th>Quantity</th><th>Received Date</th></tr></thead>
                  <tbody>{(detail?.kanbans ?? []).length === 0 ?
                    <tr><td colSpan={5}><div className="table-empty">No in-stock Kanban available.</div></td></tr> :
                    detail?.kanbans.map(kanban => <tr key={kanban.kanbanLotId}>
                      <td>{kanban.kanbanId}</td><td>{kanban.deliveryNoteNumber}</td><td>{kanban.poNumber}</td>
                      <td>{formatQuantity(kanban.quantity)} {kanban.baseUnitCode}</td><td>{kanban.receivedDate.slice(0, 10)}</td>
                    </tr>)}</tbody>
                </table>}
          </div>
        </div>
      </> : null}
    </section>
  );
}
