import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ModuleIndex } from './module-index';

describe('ModuleIndex', () => {
  it('renders compact controls, columns, and an honest empty state', () => {
    render(
      <ModuleIndex
        title="Suppliers"
        description="Supplier master"
        actionLabel="New supplier"
        columns={['Code', 'Name']}
        searchPlaceholder="Search suppliers"
        emptyMessage="No suppliers yet"
      />
    );

    expect(screen.getByRole('searchbox', { name: 'Search Suppliers' })).toHaveAttribute('placeholder', 'Search suppliers');
    expect(screen.getByRole('button', { name: 'New supplier' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Columns' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Code' }).closest('.table-frame')).toBeInTheDocument();
    expect(screen.getByText('No suppliers yet').closest('td')).toHaveClass('table-row-empty');
  });
});
