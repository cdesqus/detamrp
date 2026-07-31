export type TrendPoint = { date: string; ordered: number; received: number };

type PositionedTrendPoint = TrendPoint & {
  x: number;
  orderedY: number;
  receivedY: number;
};

function niceStep(value: number) {
  const exponent = Math.floor(Math.log10(Math.max(value, 1)));
  const magnitude = 10 ** exponent;
  const normalized = value / magnitude;
  const factor = normalized <= 1 ? 1 : normalized <= 2 ? 2 : normalized <= 5 ? 5 : 10;
  return factor * magnitude;
}

function linePath(points: PositionedTrendPoint[], key: 'orderedY' | 'receivedY') {
  return points.map((point, index) => `${index === 0 ? 'M' : 'L'}${point.x},${point[key]}`).join(' ');
}

function areaPath(line: string, points: PositionedTrendPoint[], baseline: number) {
  if (!line || points.length === 0) return '';
  return `${line} L${points.at(-1)!.x},${baseline} L${points[0].x},${baseline} Z`;
}

export function buildTrendGeometry(data: TrendPoint[], width: number, height: number) {
  const plot = { left: 40, right: width - 8, top: 12, bottom: height - 28 };
  const rawMax = Math.max(1, ...data.flatMap(point => [point.ordered, point.received]));
  const tickStep = niceStep(rawMax / 3);
  const maxValue = Math.max(tickStep * 3, Math.ceil(rawMax / tickStep) * tickStep);
  const yTicks = Array.from({ length: Math.round(maxValue / tickStep) + 1 }, (_, index) => index * tickStep);
  const x = (index: number) => data.length <= 1
    ? plot.left + (plot.right - plot.left) / 2
    : plot.left + index * ((plot.right - plot.left) / (data.length - 1));
  const y = (value: number) => plot.bottom - (value / maxValue) * (plot.bottom - plot.top);
  const points = data.map((point, index): PositionedTrendPoint => ({
    ...point,
    x: x(index),
    orderedY: y(point.ordered),
    receivedY: y(point.received),
  }));
  const labelCount = Math.min(6, data.length);
  const xLabelIndices = labelCount <= 1
    ? (data.length ? [0] : [])
    : Array.from(new Set(Array.from({ length: labelCount }, (_, index) => Math.round(index * (data.length - 1) / (labelCount - 1)))));
  const orderedLine = linePath(points, 'orderedY');
  const receivedLine = linePath(points, 'receivedY');

  return {
    plot,
    maxValue,
    yTicks,
    xLabelIndices,
    points,
    orderedLine,
    receivedLine,
    orderedArea: areaPath(orderedLine, points, plot.bottom),
    receivedArea: areaPath(receivedLine, points, plot.bottom),
  };
}
