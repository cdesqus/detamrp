import { render, screen } from '@testing-library/react';
import { expect, it, vi } from 'vitest';
import { emptyTransactionFilters, TransactionListFilters } from './transaction-list-filters';

it('renders aligned reset and apply actions with distinct visual hierarchy', () => {
  render(<TransactionListFilters
    value={emptyTransactionFilters}
    suppliers={[]}
    recordCount={4}
    loading={false}
    onChange={vi.fn()}
    onApply={vi.fn()}
    onReset={vi.fn()}
  />);

  expect(screen.getByRole('button', { name: 'Reset' })).toHaveClass('transaction-filter-button', 'transaction-filter-reset');
  expect(screen.getByRole('button', { name: 'Apply filters' })).toHaveClass('transaction-filter-button', 'transaction-filter-apply');
});
