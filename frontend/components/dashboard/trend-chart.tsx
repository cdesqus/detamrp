import { DashboardSnapshot } from './dashboard-data';

export function TrendChart({ data }: { data: DashboardSnapshot['trend'] }) {
  const width = 620, height = 180, pad = 22;
  const max = Math.max(1, ...data.flatMap(point => [point.ordered, point.received]));
  const x = (index: number) => data.length <= 1 ? width / 2 : pad + index * ((width - pad * 2) / (data.length - 1));
  const y = (value: number) => height - pad - value / max * (height - pad * 2);
  const path = (key: 'ordered' | 'received') => data.map((point, index) => `${index ? 'L' : 'M'}${x(index)},${y(point[key])}`).join(' ');
  return <div className="trend-chart">
    <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label="PO and receiving trend">
      <path className="trend-line trend-ordered" d={path('ordered')} /><path className="trend-line trend-received" d={path('received')} />
      {data.map((point, index) => <g key={point.date}>
        <circle className="trend-dot trend-ordered" cx={x(index)} cy={y(point.ordered)} r="3"><title>{point.date}: {point.ordered} ordered</title></circle>
        <circle className="trend-dot trend-received" cx={x(index)} cy={y(point.received)} r="3"><title>{point.date}: {point.received} received</title></circle>
      </g>)}
    </svg>
    <div className="chart-legend"><span><i className="legend-ordered" />Ordered</span><span><i className="legend-received" />Received</span></div>
  </div>;
}
