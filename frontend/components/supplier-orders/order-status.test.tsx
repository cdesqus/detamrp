import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { OrderStatusBadge } from './order-status';

describe('OrderStatusBadge', () => {
  it.each([
    ['DRAFT', 'Draft', 'neutral'],
    ['PENDING_APPROVAL', 'Pending Approval', 'amber'],
    ['APPROVED', 'Approved', 'green'],
    ['PARTIALLY_RECEIVED', 'Partially Received', 'blue'],
    ['FULLY_RECEIVED', 'Fully Received', 'dark-green'],
    ['REJECTED', 'Rejected', 'red'],
    ['CANCELLED', 'Cancelled', 'red'],
  ])('maps %s to an accessible %s badge', (status, label, tone) => {
    render(<OrderStatusBadge status={status} />);
    expect(screen.getByText(label)).toHaveClass(`status-pill--${tone}`);
  });
});
