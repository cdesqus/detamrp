'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { ReactNode, useEffect, useState } from 'react';
import { navigationGroups } from './navigation';

type CurrentUser = { username: string; displayName: string; permissions: string[] };
const storageKey = 'order-stock.sidebar-collapsed';

export function AppShell({ title, children }: { title: string; children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [collapsed, setCollapsed] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);

  useEffect(() => {
    setCollapsed(localStorage.getItem(storageKey) === 'true');
    fetch('/api/auth/me', { credentials: 'include' }).then(async response => {
      if (!response.ok) { router.replace('/login'); return; }
      const payload = await response.json() as { user: CurrentUser };
      setUser(payload.user);
    });
  }, [router]);

  function toggleSidebar() {
    const next = !collapsed;
    setCollapsed(next);
    localStorage.setItem(storageKey, String(next));
  }

  if (!user) return <main className="loading-page">Loading...</main>;

  return (
    <div className={`app-shell${collapsed ? ' sidebar-collapsed' : ''}${drawerOpen ? ' drawer-open' : ''}`}>
      <button className="drawer-scrim" aria-label="Close navigation" onClick={() => setDrawerOpen(false)} />
      <aside aria-label="Main navigation">
        <div className="sidebar-brand"><span>OS</span><b>Order Stock</b></div>
        <nav>
          {navigationGroups.map((group, groupIndex) => (
            <div className="nav-group" key={group.label ?? groupIndex}>
              {group.label ? <p>{group.label}</p> : null}
              {group.items.map(item => {
                const active = pathname === item.href || (item.href !== '/dashboard' && pathname.startsWith(`${item.href}/`));
                return (
                  <Link href={item.href} key={item.href} aria-current={active ? 'page' : undefined} title={collapsed ? item.label : undefined} onClick={() => setDrawerOpen(false)}>
                    <span className="nav-icon" aria-hidden="true">{item.icon}</span><span className="nav-label">{item.label}</span>
                  </Link>
                );
              })}
            </div>
          ))}
        </nav>
      </aside>
      <section className="app-content">
        <header>
          <div className="header-start">
            <button className="desktop-sidebar-toggle" aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'} aria-expanded={!collapsed} onClick={toggleSidebar}>{collapsed ? '›' : '‹'}</button>
            <button className="mobile-sidebar-toggle" aria-label="Open navigation" onClick={() => setDrawerOpen(true)}>☰</button>
            <span>{title}</span>
          </div>
          <span>{user.displayName}</span>
        </header>
        <main>{children}</main>
      </section>
    </div>
  );
}
