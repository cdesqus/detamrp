import { describe, expect, it } from 'vitest';
import { buildTrendGeometry } from './trend-chart-geometry';

describe('buildTrendGeometry', () => {
  it('keeps one-day line and area paths finite and inside the plot', () => {
    const result = buildTrendGeometry([{ date: '2026-07-24', ordered: 5, received: 3 }], 720, 240);

    expect(result.orderedLine).not.toMatch(/NaN|Infinity/);
    expect(result.receivedLine).not.toMatch(/NaN|Infinity/);
    expect(result.orderedArea).not.toMatch(/NaN|Infinity/);
    expect(result.orderedArea.endsWith('Z')).toBe(true);
    expect(result.points[0].x).toBe(376);
  });

  it('selects at most six date labels including the first and last day', () => {
    const data = Array.from({ length: 30 }, (_, index) => ({
      date: `2026-07-${String(index + 1).padStart(2, '0')}`,
      ordered: index,
      received: index / 2,
    }));

    const result = buildTrendGeometry(data, 720, 240);

    expect(result.xLabelIndices.length).toBeLessThanOrEqual(6);
    expect(result.xLabelIndices[0]).toBe(0);
    expect(result.xLabelIndices.at(-1)).toBe(29);
  });

  it('creates readable rounded Y ticks from zero through the data ceiling', () => {
    const result = buildTrendGeometry([
      { date: '2026-07-23', ordered: 7, received: 2 },
      { date: '2026-07-24', ordered: 13, received: 9 },
    ], 720, 240);

    expect(result.maxValue).toBe(15);
    expect(result.yTicks).toEqual([0, 5, 10, 15]);
  });
});
