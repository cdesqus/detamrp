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
  const { container } = render(<CompanySettings />);

  const input = await screen.findByRole('textbox', { name: 'Company Name' });
  expect(input).toHaveValue('Our Company');
  expect(screen.getByRole('group', { name: 'Company information' })).toContainElement(input);
  expect(screen.getByRole('group', { name: 'Company Logo' })).toBeInTheDocument();
  expect(screen.getByRole('group', { name: 'Login Background' })).toBeInTheDocument();
  expect(screen.getByRole('form', { name: 'Company settings' })).toHaveClass('company-settings-shell');
  expect(screen.getByRole('region', { name: 'Brand assets' })).toHaveClass('company-assets-section');
  expect(screen.getByRole('group', { name: 'Company Logo' })).toHaveClass('company-asset-card');
  expect(container.querySelector('footer.company-settings-footer')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Save settings' })).toHaveClass('company-settings-save');
  await user.clear(input);
  await user.type(input, 'PT Buyer Indonesia');
  await user.click(screen.getByRole('button', { name: 'Save settings' }));

  await waitFor(() => expect(fetchMock).toHaveBeenLastCalledWith('/api/settings/company', expect.objectContaining({
    method: 'PUT',
    body: JSON.stringify({ companyName: 'PT Buyer Indonesia' })
  })));
  expect(await screen.findByRole('status')).toHaveTextContent('Company settings saved.');
});

it('uploads and resets company branding media', async () => {
  const fetchMock = vi.fn()
    .mockResolvedValueOnce({ ok: true, json: async () => ({ companyName: 'Our Company', logoUrl: null, loginBackgroundUrl: null }) })
    .mockResolvedValueOnce({ ok: true, json: async () => ({ companyName: 'Our Company', logoUrl: '/api/public/branding/logo', loginBackgroundUrl: null }) })
    .mockResolvedValueOnce({ ok: true, json: async () => ({ companyName: 'Our Company', logoUrl: null, loginBackgroundUrl: null }) });
  vi.stubGlobal('fetch', fetchMock);
  const user = userEvent.setup();
  render(<CompanySettings />);

  const logo = await screen.findByLabelText('Company Logo');
  await user.upload(logo, new File(['image'], 'logo.png', { type: 'image/png' }));
  await user.click(screen.getByRole('button', { name: 'Upload logo' }));
  expect(fetchMock).toHaveBeenCalledWith('/api/settings/company/logo', expect.objectContaining({ method: 'PUT' }));

  await user.click(await screen.findByRole('button', { name: 'Reset logo' }));
  expect(fetchMock).toHaveBeenCalledWith('/api/settings/company/logo', expect.objectContaining({ method: 'DELETE' }));
});
