import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DashboardChart } from './dashboard-chart';
import { emptyDashboardSnapshot } from './dashboard-data';

describe('DashboardChart', () => {
  it('shows an honest empty state when there are no transactions', () => {
    render(<DashboardChart title="PO & Receiving Trend" empty><svg aria-label="PO trend chart" /></DashboardChart>);
    expect(screen.getByRole('heading', { name: 'PO & Receiving Trend' })).toBeInTheDocument();
    expect(screen.getByText('No transaction data yet')).toBeInTheDocument();
    expect(screen.queryByLabelText('PO trend chart')).not.toBeInTheDocument();
  });

  it('defines zero metrics and empty real-data series', () => {
    expect(Object.values(emptyDashboardSnapshot.metrics).every(value => value === 0)).toBe(true);
    expect(emptyDashboardSnapshot.trend).toHaveLength(0);
    expect(emptyDashboardSnapshot.poStatus).toHaveLength(0);
    expect(emptyDashboardSnapshot.outstandingBySupplier).toHaveLength(0);
    expect(emptyDashboardSnapshot.activities).toHaveLength(0);
  });
});
