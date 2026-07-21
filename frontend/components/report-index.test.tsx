import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ReportIndex } from './report-index';

describe('ReportIndex', () => {
  it('offers real-data filters including supplier and an honest empty state', () => {
    render(<ReportIndex />);
    expect(screen.getByLabelText('Supplier')).toBeInTheDocument();
    expect(screen.getByLabelText('Raw Material')).toBeInTheDocument();
    expect(screen.getByLabelText('PO Reference')).toBeInTheDocument();
    expect(screen.getByText('Belum ada data report')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Export' })).toBeDisabled();
  });
});
