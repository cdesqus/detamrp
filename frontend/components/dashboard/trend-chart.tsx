'use client';

import { KeyboardEvent, useState } from 'react';
import { DashboardSnapshot } from './dashboard-data';
import { buildTrendGeometry, tooltipPlacement } from './trend-chart-geometry';

const width = 720;
const height = 220;

function shortDate(value: string) {
  const [, month, day] = value.split('-');
  return month && day ? `${day}/${month}` : value;
}

export function TrendChart({ data }: { data: DashboardSnapshot['trend'] }) {
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const geometry = buildTrendGeometry(data, width, height);
  const active = activeIndex === null ? null : geometry.points[activeIndex];
  const activePlacement = activeIndex === null ? 'center' : tooltipPlacement(activeIndex, data.length);
  const hitWidth = geometry.points.length <= 1
    ? geometry.plot.right - geometry.plot.left
    : (geometry.plot.right - geometry.plot.left) / (geometry.points.length - 1);

  function navigate(event: KeyboardEvent<SVGRectElement>, index: number) {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
    event.preventDefault();
    const direction = event.key === 'ArrowRight' ? 1 : -1;
    setActiveIndex(Math.min(data.length - 1, Math.max(0, index + direction)));
  }

  return <div className="trend-chart">
    <div className="trend-chart-canvas" onPointerLeave={() => setActiveIndex(null)}>
      <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label="PO and receiving trend">
        <defs>
          <linearGradient id="trend-ordered-area" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="#2563eb" stopOpacity=".18" />
            <stop offset="1" stopColor="#2563eb" stopOpacity="0" />
          </linearGradient>
          <linearGradient id="trend-received-area" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="#16a34a" stopOpacity=".16" />
            <stop offset="1" stopColor="#16a34a" stopOpacity="0" />
          </linearGradient>
        </defs>

        <g className="trend-axis" aria-hidden="true">
          {geometry.yTicks.map(tick => {
            const y = geometry.plot.bottom - (tick / geometry.maxValue) * (geometry.plot.bottom - geometry.plot.top);
            return <g key={tick}><line x1={geometry.plot.left} x2={geometry.plot.right} y1={y} y2={y} /><text x={geometry.plot.left - 9} y={y + 3}>{tick}</text></g>;
          })}
          {geometry.xLabelIndices.map(index => <text key={data[index].date} className="trend-axis-date" x={geometry.points[index].x} y={height - 7}>{shortDate(data[index].date)}</text>)}
        </g>

        <path className="trend-series-area trend-series-area--ordered" d={geometry.orderedArea} />
        <path className="trend-series-area trend-series-area--received" d={geometry.receivedArea} />
        <path className="trend-series-line trend-series-line--ordered" d={geometry.orderedLine} />
        <path className="trend-series-line trend-series-line--received" d={geometry.receivedLine} />

        {active ? <line aria-hidden="true" className="trend-crosshair" x1={active.x} x2={active.x} y1={geometry.plot.top} y2={geometry.plot.bottom} /> : null}

        {geometry.points.map((point, index) => <g key={point.date}>
          <circle className={`trend-series-dot trend-series-dot--ordered${activeIndex === index ? ' is-active' : ''}`} cx={point.x} cy={point.orderedY} r={activeIndex === index ? 4 : 2.5}>
            <title>{point.date}: {point.ordered} ordered</title>
          </circle>
          <circle className={`trend-series-dot trend-series-dot--received${activeIndex === index ? ' is-active' : ''}`} cx={point.x} cy={point.receivedY} r={activeIndex === index ? 4 : 2.5}>
            <title>{point.date}: {point.received} received</title>
          </circle>
          <rect
            className="trend-hit-area"
            x={point.x - hitWidth / 2}
            y={geometry.plot.top}
            width={hitWidth}
            height={geometry.plot.bottom - geometry.plot.top}
            rx="2"
            role="button"
            tabIndex={0}
            aria-label={`${point.date}: ${point.ordered} ordered, ${point.received} received`}
            onFocus={() => setActiveIndex(index)}
            onPointerEnter={() => setActiveIndex(index)}
            onKeyDown={event => navigate(event, index)}
          />
        </g>)}
      </svg>

      {active ? <div
        className="trend-tooltip"
        role="status"
        data-placement={activePlacement}
        style={{
          left: `${active.x / width * 100}%`,
          top: `${Math.max(48, Math.min(active.orderedY, active.receivedY) / height * 100 - 3)}%`,
        }}
      >
        <strong>{active.date}</strong>
        <span className="trend-tooltip-row"><i className="legend-ordered" /><span>Ordered</span>{' '}<b>{active.ordered}</b></span>
        <span className="trend-tooltip-row"><i className="legend-received" /><span>Received</span>{' '}<b>{active.received}</b></span>
      </div> : null}
    </div>
    <div className="chart-legend" aria-hidden="true"><span><i className="legend-ordered" />Ordered</span><span><i className="legend-received" />Received</span></div>
  </div>;
}
