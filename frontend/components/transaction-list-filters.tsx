'use client';

type Supplier = { id: string; code?: string; name: string };
export type TransactionFilters = { search: string; supplierId: string; createdFrom: string; createdTo: string };
export const emptyTransactionFilters: TransactionFilters = { search: '', supplierId: '', createdFrom: '', createdTo: '' };

export function TransactionListFilters({ value, suppliers, recordCount, loading, showSearch = false, onChange, onApply, onReset }: {
  value: TransactionFilters; suppliers: Supplier[]; recordCount: number; loading: boolean; showSearch?: boolean;
  onChange: (value: TransactionFilters) => void; onApply: () => void; onReset: () => void;
}) {
  const invalidRange = Boolean(value.createdFrom && value.createdTo && value.createdFrom > value.createdTo);
  const update = (key: keyof TransactionFilters, next: string) => onChange({ ...value, [key]: next });
  return <div className="transaction-filter-panel" aria-label="Table filters">
    <div className="transaction-filter-fields">
      {showSearch && <label className="transaction-filter-search">Search<input aria-label="Search Supplier Orders" type="search" value={value.search} onChange={event => update('search', event.target.value)} placeholder="PO number or notes" /></label>}
      <label>Supplier<select aria-label="Supplier filter" value={value.supplierId} onChange={event => update('supplierId', event.target.value)}><option value="">All suppliers</option>{suppliers.map(supplier => <option key={supplier.id} value={supplier.id}>{supplier.code ? `${supplier.code} — ` : ''}{supplier.name}</option>)}</select></label>
      <label>Created from<input aria-label="Created from" type="date" value={value.createdFrom} onChange={event => update('createdFrom', event.target.value)} /></label>
      <label>Created to<input aria-label="Created to" type="date" value={value.createdTo} onChange={event => update('createdTo', event.target.value)} /></label>
    </div>
    <div className="transaction-filter-actions">
      <span className="transaction-filter-count">{loading ? 'Refreshing…' : `${recordCount} records`}</span>
      <button type="button" className="table-action" onClick={onReset}>Reset</button>
      <button type="button" className="primary-button transaction-filter-apply" disabled={invalidRange} onClick={onApply}>Apply filters</button>
    </div>
    {invalidRange && <p className="transaction-filter-error" role="alert">Created to must be on or after Created from.</p>}
  </div>;
}
