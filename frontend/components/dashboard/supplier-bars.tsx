import { DashboardSnapshot } from './dashboard-data';

export function SupplierBars({ data }: { data: DashboardSnapshot['outstandingBySupplier'] }) {
  const max = Math.max(1, ...data.map(item => item.kanban));
  return <div className="supplier-bars" aria-label="Outstanding Kanban by supplier">{data.map(item => <div className="supplier-bar" key={item.supplier}>
    <div><span title={item.supplier}>{item.supplier}</span><strong>{item.kanban}</strong></div><i><b style={{ width: `${item.kanban / max * 100}%` }} /></i>
  </div>)}</div>;
}
