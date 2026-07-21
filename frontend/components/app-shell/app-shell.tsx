'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { ReactNode, useEffect, useState } from 'react';
import { Icon } from '../icons';
import { NotificationCenter } from '../notifications/notification-center';
import { emptyNotificationSnapshot } from '../notifications/notification-data';
import { navigationGroups } from './navigation';

type CurrentUser = { username: string; displayName: string; permissions: string[] };
const storageKey = 'order-stock.sidebar-collapsed';

export function AppShell({ title, children }: { title: string; children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [collapsed, setCollapsed] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(pathname.startsWith('/settings'));

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
        <div className="sidebar-brand"><span>OS</span><b>Order Stock</b><button className="sidebar-collapse" aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'} aria-expanded={!collapsed} onClick={toggleSidebar}><Icon name={collapsed ? 'chevron-right' : 'chevron-left'} /></button></div>
        <nav>
          {navigationGroups.map((group, groupIndex) => (
            <div className="nav-group" key={group.label ?? groupIndex}>
              {group.collapsible ? <button className="nav-group-toggle" aria-expanded={settingsOpen} onClick={() => setSettingsOpen(value => !value)}><Icon name={group.icon ?? 'settings'} /><span className="nav-label">{group.label}</span><Icon name={settingsOpen ? 'chevron-left' : 'chevron-right'} /></button> : group.label ? <p>{group.label}</p> : null}
              {(!group.collapsible || settingsOpen) && group.items.map(item => {
                const active = pathname === item.href || (item.href !== '/dashboard' && pathname.startsWith(`${item.href}/`));
                return (
                  <Link href={item.href} key={item.href} aria-current={active ? 'page' : undefined} title={collapsed ? item.label : undefined} onClick={() => setDrawerOpen(false)}>
                    <span className="nav-icon"><Icon name={item.icon} /></span><span className="nav-label">{item.label}</span>
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
            <button className="mobile-sidebar-toggle" aria-label="Open navigation" onClick={() => setDrawerOpen(true)}><Icon name="menu" /></button>
            <span>{title}</span>
          </div>
          <div className="header-end"><NotificationCenter items={emptyNotificationSnapshot.items} /><span>{user.displayName}</span></div>
        </header>
        <main>{children}</main>
      </section>
    </div>
  );
}
