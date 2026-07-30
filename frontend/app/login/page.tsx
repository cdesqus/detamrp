'use client';

import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { LoginForm } from '../../components/login-form';

export default function LoginPage() {
  const router = useRouter();
  const [branding, setBranding] = useState({ companyName: 'DETA MRP', logoUrl: null as string | null, loginBackgroundUrl: null as string | null });
  useEffect(() => { fetch('/api/public/branding').then(async response => {
    if (response.ok) setBranding(await response.json());
  }).catch(() => undefined); }, []);
  async function login(username: string, password: string) {
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
    if (!response.ok) throw new Error('Invalid username or password');
    const requested = new URLSearchParams(window.location.search).get('next');
    router.push(requested?.startsWith('/') && !requested.startsWith('//') ? requested : '/dashboard');
    router.refresh();
  }

  return (
    <main className="login-page" style={branding.loginBackgroundUrl ? { backgroundImage: `linear-gradient(rgb(9 9 11 / 58%), rgb(9 9 11 / 58%)), url(/api${branding.loginBackgroundUrl})` } : undefined}>
      <section className="login-panel">
        {branding.logoUrl ? <img className="login-logo" src={`/api${branding.logoUrl}`} alt={`${branding.companyName} logo`} /> : <div className="brand-mark">DM</div>}
        <h1>{branding.companyName}</h1>
        {branding.companyName !== 'DETA MRP' && <strong className="product-name">DETA MRP</strong>}
        <p>Logistics & Production Control</p>
        <LoginForm onLogin={login} />
      </section>
    </main>
  );
}
