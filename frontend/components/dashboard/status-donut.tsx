import { DashboardSnapshot } from './dashboard-data';

const colors = ['#2563eb', '#16a34a', '#d97706', '#dc2626', '#7c3aed', '#64748b', '#0891b2'];
const label = (value: string) => value.replaceAll('_', ' ').toLowerCase().replace(/\b\w/g, letter => letter.toUpperCase());

export function StatusDonut({ data }: { data: DashboardSnapshot['poStatus'] }) {
  const total = data.reduce((sum, item) => sum + item.count, 0);
  let offset = 0;
  return <div className="donut-wrap" aria-label="PO status distribution">
    <svg viewBox="0 0 120 120" role="img">
      <circle className="donut-track" cx="60" cy="60" r="42" />
      {data.map((item, index) => {
        const size = item.count / total * 264;
        const segment = <circle key={item.status} className="donut-segment" cx="60" cy="60" r="42" stroke={colors[index % colors.length]} strokeDasharray={`${size} ${264 - size}`} strokeDashoffset={-offset}><title>{label(item.status)}: {item.count}</title></circle>;
        offset += size; return segment;
      })}
      <text x="60" y="57" textAnchor="middle">{total}</text><text className="donut-caption" x="60" y="72" textAnchor="middle">POs</text>
    </svg>
    <div className="donut-legend">{data.map((item, index) => <span key={item.status}><i style={{ background: colors[index % colors.length] }} />{label(item.status)} <b>{item.count}</b></span>)}</div>
  </div>;
}
