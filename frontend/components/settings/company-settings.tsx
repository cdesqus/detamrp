'use client';

import { FormEvent, useEffect, useState } from 'react';

export function CompanySettings() {
  const [companyName, setCompanyName] = useState('');
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    fetch('/api/settings/company', { credentials: 'include' })
      .then(async response => {
        if (!response.ok) throw new Error();
        const value = await response.json() as { companyName: string };
        setCompanyName(value.companyName);
      })
      .catch(() => setError('Company settings could not be loaded.'));
  }, []);

  async function save(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError('');
    setMessage('');
    const response = await fetch('/api/settings/company', {
      method: 'PUT',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ companyName })
    });
    const payload = await response.json().catch(() => ({})) as { companyName?: string; message?: string; fields?: Record<string, string> };
    setBusy(false);
    if (!response.ok) {
      setError(payload.fields?.companyName ?? payload.message ?? 'Company settings could not be saved.');
      return;
    }
    setCompanyName(payload.companyName ?? companyName);
    setMessage('Company settings saved.');
  }

  return <section className="settings-page">
    <div className="page-title-row"><div><h1>Company Settings</h1><p className="muted">Company identity printed on Purchase Orders, Delivery Notes, and Kanban Cards.</p></div></div>
    <form className="settings-card" onSubmit={save}>
      <div className="crud-fields"><label>Company Name<input aria-label="Company Name" value={companyName} maxLength={160} required onChange={event => setCompanyName(event.target.value)} /></label></div>
      {error && <p className="form-error" role="alert">{error}</p>}
      {message && <p role="status">{message}</p>}
      <div className="settings-actions"><button className="primary-button" disabled={busy}>{busy ? 'Saving...' : 'Save settings'}</button></div>
    </form>
  </section>;
}
