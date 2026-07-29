import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, it, vi } from 'vitest';
import { CompanySettings } from './company-settings';

it('loads and saves the company name', async () => {
  const fetchMock = vi.fn()
    .mockResolvedValueOnce({ ok: true, json: async () => ({ companyName: 'Our Company' }) })
    .mockResolvedValueOnce({ ok: true, json: async () => ({ companyName: 'PT Buyer Indonesia' }) });
  vi.stubGlobal('fetch', fetchMock);
  const user = userEvent.setup();
  render(<CompanySettings />);

  const input = await screen.findByRole('textbox', { name: 'Company Name' });
  expect(input).toHaveValue('Our Company');
  await user.clear(input);
  await user.type(input, 'PT Buyer Indonesia');
  await user.click(screen.getByRole('button', { name: 'Save settings' }));

  await waitFor(() => expect(fetchMock).toHaveBeenLastCalledWith('/api/settings/company', expect.objectContaining({
    method: 'PUT',
    body: JSON.stringify({ companyName: 'PT Buyer Indonesia' })
  })));
  expect(await screen.findByRole('status')).toHaveTextContent('Company settings saved.');
});
