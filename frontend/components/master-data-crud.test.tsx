import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MasterDataCrud } from './master-data-crud';

describe('MasterDataCrud drawer', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ items: [], total: 0 })
    }));
  });

  it('opens an interactive modal dialog instead of a navigation aside', async () => {
    const user = userEvent.setup();
    render(<MasterDataCrud title="Suppliers" description="Supplier master" endpoint="/master-data/suppliers" singular="supplier" searchPlaceholder="Search" initial={{ code: '', active: true }} columns={[{ key: 'code', label: 'Code' }]} fields={[{ key: 'code', label: 'Supplier ID', required: true }]} />);

    await user.click(screen.getByRole('button', { name: 'New supplier' }));

    const dialog = screen.getByRole('dialog', { name: 'New supplier' });
    const input = screen.getByRole('textbox', { name: /Supplier ID/ });
    expect(dialog).toHaveAttribute('aria-modal', 'true');
    await user.type(input, 'SUP-001');
    expect(input).toHaveValue('SUP-001');
  });
});
