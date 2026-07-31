import { render, screen } from '@testing-library/react';
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

  it('renders status and supplier values as accessible summaries', () => {
    render(<><StatusDonut data={[{ status: 'APPROVED', count: 2 }]} /><SupplierBars data={[{ supplier: 'PT Example', kanban: 4 }]} /></>);
    expect(screen.getByLabelText('PO status distribution')).toBeInTheDocument();
    expect(screen.getByText('Approved: 2')).toBeInTheDocument();
    expect(screen.getByLabelText('Outstanding Kanban by supplier')).toHaveTextContent('PT Example');
    expect(screen.getByLabelText('Outstanding Kanban by supplier')).toHaveTextContent('4');
  });
});
