import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { StatusDonut } from './status-donut';
import { SupplierBars } from './supplier-bars';
import { TrendChart } from './trend-chart';

describe('dashboard charts', () => {
  it('renders a finite one-day trend with exact values', () => {
    const { container } = render(<TrendChart data={[{ date: '2026-07-24', ordered: 5, received: 3 }]} />);
    expect(screen.getByLabelText('PO and receiving trend')).toBeInTheDocument();
    expect(container.innerHTML).not.toContain('NaN');
    expect(screen.getByText('2026-07-24: 5 ordered')).toBeInTheDocument();
    expect(screen.getByText('2026-07-24: 3 received')).toBeInTheDocument();
  });

  it('shows both series in one keyboard-accessible date tooltip', async () => {
    const user = userEvent.setup();
    const { container } = render(<TrendChart data={[
      { date: '2026-07-23', ordered: 2, received: 1 },
      { date: '2026-07-24', ordered: 5, received: 3 },
    ]} />);

    await user.tab();

    expect(screen.getByRole('status')).toHaveTextContent('2026-07-23');
    expect(screen.getByRole('status')).toHaveTextContent('Ordered 2');
    expect(screen.getByRole('status')).toHaveTextContent('Received 1');
    expect(container.querySelectorAll('path.trend-series-line')).toHaveLength(2);
    expect(container.querySelectorAll('path.trend-series-area')).toHaveLength(2);
  });

  it('keeps exact values paired with their series and anchors edge tooltips', async () => {
    const user = userEvent.setup();
    render(<TrendChart data={[
      { date: '2026-07-28', ordered: 0, received: 20 },
      { date: '2026-07-31', ordered: 21, received: 0 },
    ]} />);

    await user.tab();
    let tooltip = screen.getByRole('status');
    expect(tooltip).toHaveAttribute('data-placement', 'start');
    expect(within(tooltip).getByText('Ordered').closest('.trend-tooltip-row')).toHaveTextContent('Ordered 0');
    expect(within(tooltip).getByText('Received').closest('.trend-tooltip-row')).toHaveTextContent('Received 20');

    await user.tab();
    tooltip = screen.getByRole('status');
    expect(tooltip).toHaveAttribute('data-placement', 'end');
    expect(within(tooltip).getByText('Ordered').closest('.trend-tooltip-row')).toHaveTextContent('Ordered 21');
    expect(within(tooltip).getByText('Received').closest('.trend-tooltip-row')).toHaveTextContent('Received 0');
  });

  it('renders status and supplier values as accessible summaries', () => {
    render(<><StatusDonut data={[{ status: 'APPROVED', count: 2 }]} /><SupplierBars data={[{ supplier: 'PT Example', kanban: 4 }]} /></>);
    expect(screen.getByLabelText('PO status distribution')).toBeInTheDocument();
    expect(screen.getByText('Approved: 2')).toBeInTheDocument();
    expect(screen.getByLabelText('Outstanding Kanban by supplier')).toHaveTextContent('PT Example');
    expect(screen.getByLabelText('Outstanding Kanban by supplier')).toHaveTextContent('4');
  });
});
