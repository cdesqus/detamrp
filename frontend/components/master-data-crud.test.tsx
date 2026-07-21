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
    expect(dialog).toHaveClass('crud-modal', 'crud-modal--compact');
    await user.type(input, 'SUP-001');
    expect(input).toHaveValue('SUP-001');
  });

  it('uses the same centered modal for edit', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ items: [{ id: '1', code: 'SUP-001', active: true }], total: 1 }) }));
    const user = userEvent.setup();
    render(<MasterDataCrud title="Suppliers" description="Supplier master" endpoint="/master-data/suppliers" singular="supplier" searchPlaceholder="Search" initial={{ code: '', active: true }} columns={[{ key: 'code', label: 'Code' }]} fields={[{ key: 'code', label: 'Supplier ID', required: true }]} />);
    await user.click(await screen.findByRole('button', { name: 'Edit' }));
    expect(screen.getByRole('dialog', { name: 'Edit supplier' })).toHaveClass('crud-modal', 'crud-modal--compact');
  });
});
