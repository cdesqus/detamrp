'use client';

import { useRouter } from 'next/navigation';
import { LoginForm } from '../../components/login-form';

export default function LoginPage() {
  const router = useRouter();
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
    <main className="login-page">
      <section className="login-panel">
        <div className="brand-mark">OS</div>
        <h1>Order Stock</h1>
        <p>Logistics & Production Control</p>
        <LoginForm onLogin={login} />
      </section>
    </main>
  );
}
