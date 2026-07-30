'use client';

import { FormEvent, useEffect, useState } from 'react';

type CompanyConfig = { companyName: string; logoUrl: string | null; loginBackgroundUrl: string | null };
type MediaKind = 'logo' | 'login-background';

export function CompanySettings() {
  const [config, setConfig] = useState<CompanyConfig>({ companyName: '', logoUrl: null, loginBackgroundUrl: null });
  const [files, setFiles] = useState<Partial<Record<MediaKind, File>>>({});
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    fetch('/api/settings/company', { credentials: 'include' })
      .then(async response => {
        if (!response.ok) throw new Error();
        setConfig(await response.json() as CompanyConfig);
      })
      .catch(() => setError('Company settings could not be loaded.'));
  }, []);

  async function request(url: string, init: RequestInit, success: string) {
    setBusy(true); setError(''); setMessage('');
    const response = await fetch(url, { credentials: 'include', ...init });
    const payload = await response.json().catch(() => ({})) as CompanyConfig & { message?: string; fields?: Record<string, string> };
    setBusy(false);
    if (!response.ok) {
      setError(Object.values(payload.fields ?? {})[0] ?? payload.message ?? 'Company settings could not be saved.');
      return false;
    }
    setConfig(payload); setMessage(success);
    return true;
  }

  async function save(event: FormEvent) {
    event.preventDefault();
    await request('/api/settings/company', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ companyName: config.companyName }) }, 'Company settings saved.');
  }
  async function upload(kind: MediaKind) {
    const file = files[kind];
    if (!file) { setError('Select an image first.'); return; }
    const data = new FormData(); data.append('file', file);
    if (await request(`/api/settings/company/${kind}`, { method: 'PUT', body: data }, `${kind === 'logo' ? 'Logo' : 'Login background'} updated.`)) {
      setFiles(value => ({ ...value, [kind]: undefined }));
    }
  }
  async function reset(kind: MediaKind) {
    await request(`/api/settings/company/${kind}`, { method: 'DELETE' }, `${kind === 'logo' ? 'Logo' : 'Login background'} reset.`);
  }
  function choose(kind: MediaKind, file?: File) {
    if (!file) { setFiles(value => ({ ...value, [kind]: undefined })); return; }
    if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) {
      setError('Use a PNG, JPEG, or WebP image.'); return;
    }
    const max = kind === 'logo' ? 2 * 1024 * 1024 : 5 * 1024 * 1024;
    if (file.size > max) {
      setError(`${kind === 'logo' ? 'Logo' : 'Login background'} exceeds the ${kind === 'logo' ? '2' : '5'} MB limit.`); return;
    }
    setError(''); setFiles(value => ({ ...value, [kind]: file }));
  }

  return <section className="settings-page">
    <div className="page-title-row"><div><h1>Company Settings</h1><p className="muted">Identity used in DETA MRP, login, Purchase Orders, Delivery Notes, and Kanban Cards.</p></div></div>
    <form className="settings-card company-branding-card" onSubmit={save}>
      <div className="crud-fields"><label>Company Name<input aria-label="Company Name" value={config.companyName} maxLength={160} required onChange={event => setConfig(value => ({ ...value, companyName: event.target.value }))} /></label></div>
      <BrandingField title="Company Logo" hint="PNG, JPEG, or WebP — max 2 MB" kind="logo" url={config.logoUrl} file={files.logo} busy={busy} onFile={file => choose('logo', file)} onUpload={upload} onReset={reset} />
      <BrandingField title="Login Background" hint="PNG, JPEG, or WebP — max 5 MB" kind="login-background" url={config.loginBackgroundUrl} file={files['login-background']} busy={busy} onFile={file => choose('login-background', file)} onUpload={upload} onReset={reset} />
      {error && <p className="form-error" role="alert">{error}</p>}
      {message && <p role="status">{message}</p>}
      <div className="settings-actions"><button className="primary-button" disabled={busy}>{busy ? 'Saving...' : 'Save settings'}</button></div>
    </form>
  </section>;
}

function BrandingField({ title, hint, kind, url, file, busy, onFile, onUpload, onReset }: {
  title: string; hint: string; kind: MediaKind; url: string | null; file?: File; busy: boolean;
  onFile: (file?: File) => void; onUpload: (kind: MediaKind) => void; onReset: (kind: MediaKind) => void;
}) {
  const [localPreview, setLocalPreview] = useState<string | null>(null);
  useEffect(() => {
    if (!file || typeof URL.createObjectURL !== 'function') { setLocalPreview(null); return; }
    const value = URL.createObjectURL(file); setLocalPreview(value);
    return () => URL.revokeObjectURL(value);
  }, [file]);
  const preview = localPreview ?? (url ? `/api${url}` : null);
  const action = kind === 'logo' ? 'logo' : 'background';
  return <div className="branding-field">
    <div className={`branding-preview ${kind}`}>{preview ? <img src={preview} alt={`${title} preview`} /> : <span>DETA MRP</span>}</div>
    <div className="branding-controls"><strong>{title}</strong><small>{hint}</small>
      <input aria-label={title} type="file" accept="image/png,image/jpeg,image/webp" onChange={event => onFile(event.target.files?.[0])} />
      <div><button type="button" disabled={busy || !file} onClick={() => onUpload(kind)}>Upload {action}</button>{url && <button type="button" disabled={busy} onClick={() => onReset(kind)}>Reset {action}</button>}</div>
    </div>
  </div>;
}
