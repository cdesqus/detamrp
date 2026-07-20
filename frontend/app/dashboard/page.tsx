'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';

type CurrentUser = { username: string; displayName: string; permissions: string[] };

export default function DashboardPage() {
  const router = useRouter();
  const [user, setUser] = useState<CurrentUser | null>(null);
  useEffect(() => {
    fetch('/api/auth/me', { credentials: 'include' }).then(async response => {
      if (!response.ok) { router.replace('/login'); return; }
      const payload = await response.json() as { user: CurrentUser };
      setUser(payload.user);
    });
  }, [router]);

  if (!user) return <main className="loading-page">Loading…</main>;
  return (
    <div className="app-shell">
      <aside>
        <div className="sidebar-brand"><span>OS</span> Order Stock</div>
        <nav>
          <a className="active" href="/dashboard">Dashboard</a>
          <p>Procurement</p><a href="#">Supplier Orders</a><a href="#">Approval Inbox</a>
          <p>Logistics</p><a href="#">Delivery Notes</a><a href="#">Receiving</a><a href="#">Outgoing Material</a>
          <p>Data Master</p><a href="#">Measurements</a><a href="#">Suppliers</a><a href="#">Raw Materials</a>
        </nav>
      </aside>
      <section className="app-content">
        <header><span>Dashboard</span><span>{user.displayName}</span></header>
        <main>
          <h1>Dashboard</h1>
          <p className="muted">Foundation ready. Data Master is being implemented.</p>
          <div className="metric-grid">
            <article><span>Pending Approval</span><strong>0</strong></article>
            <article><span>Expected Today</span><strong>0</strong></article>
            <article><span>Received Today</span><strong>0</strong></article>
            <article><span>Outstanding Kanban</span><strong>0</strong></article>
          </div>
        </main>
      </section>
    </div>
  );
}
