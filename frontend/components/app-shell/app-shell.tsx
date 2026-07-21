'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { createContext, ReactNode, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { Icon } from '../icons';
import { NotificationCenter } from '../notifications/notification-center';
import { NotificationItem } from '../notifications/notification-data';
import { navigationGroups } from './navigation';

export type CurrentUser = { username: string; displayName: string; permissions: string[] };
type PendingApproval = { id: string; purchaseOrderId: string; version?: number; poNumber?: string; supplierId?: string; supplierName?: string };
const CurrentUserContext = createContext<CurrentUser | null>(null);
const storageKey = 'order-stock.sidebar-collapsed';
const approvalRefreshEvent = 'purchase-order-approvals:refresh';

export function useCurrentUser() { return useContext(CurrentUserContext); }

export function AppShell({ title, children }: { title: string; children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [notificationItems, setNotificationItems] = useState<NotificationItem[]>([]);
  const [notificationTotal, setNotificationTotal] = useState(0);
  const notificationRequest = useRef<{ generation: number; controller: AbortController | null }>({ generation: 0, controller: null });
  const [collapsed, setCollapsed] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>(() => Object.fromEntries(navigationGroups.filter(group => group.collapsible && group.label).map(group => [group.label!, group.items.some(item => pathname === item.href || pathname.startsWith(`${item.href}/`))])));

  useEffect(() => {
    setCollapsed(localStorage.getItem(storageKey) === 'true');
    fetch('/api/auth/me', { credentials: 'include' }).then(async response => {
      if (!response.ok) { router.replace('/login'); return; }
      const payload = await response.json() as { user: CurrentUser };
      setUser(payload.user);
    });
  }, [router]);

  const loadApprovalNotifications = useCallback(async () => {
    notificationRequest.current.controller?.abort();
    const generation = ++notificationRequest.current.generation;
    if (!user?.permissions.includes('po.approve')) {
      setNotificationItems([]);
      setNotificationTotal(0);
      return;
    }
    const controller = new AbortController();
    notificationRequest.current.controller = controller;
    try {
      const response = await fetch('/api/purchase-order-approvals?limit=200', { credentials: 'include', signal: controller.signal });
      if (!response.ok) return;
      const payload = await response.json() as { items?: PendingApproval[]; total?: number };
      if (generation !== notificationRequest.current.generation || controller.signal.aborted) return;
      const approvals = payload.items ?? [];
      const canViewPurchaseOrders = user.permissions.includes('po.view');
      setNotificationItems(approvals.map(approval => {
        const reference = approval.poNumber ?? `PO ${approval.purchaseOrderId}`;
        return {
        id: approval.id,
        title: `${reference} awaits approval`,
        description: approval.supplierName ?? (approval.version ? `Approval request v${approval.version}` : 'Approval request is pending'),
        href: canViewPurchaseOrders ? `/supplier-orders/${approval.purchaseOrderId}` : '/approvals',
        unread: true,
        type: 'approval'
      }; }));
      setNotificationTotal(payload.total ?? approvals.length);
    } catch { /* Preserve the last good notification snapshot on refresh failure. */ }
  }, [user]);

  useEffect(() => {
    void loadApprovalNotifications();
    const refresh = () => void loadApprovalNotifications();
    window.addEventListener(approvalRefreshEvent, refresh);
    return () => {
      window.removeEventListener(approvalRefreshEvent, refresh);
      notificationRequest.current.controller?.abort();
      notificationRequest.current.generation += 1;
    };
  }, [loadApprovalNotifications]);

  function toggleSidebar() {
    const next = !collapsed;
    setCollapsed(next);
    localStorage.setItem(storageKey, String(next));
  }

  if (!user) return <main className="loading-page">Loading...</main>;

  return (
    <CurrentUserContext.Provider value={user}><div className={`app-shell${collapsed ? ' sidebar-collapsed' : ''}${drawerOpen ? ' drawer-open' : ''}`}>
      <button className="drawer-scrim" aria-label="Close navigation" onClick={() => setDrawerOpen(false)} />
      <aside aria-label="Main navigation">
        <div className="sidebar-brand"><span>OS</span><b>Order Stock</b><button className="sidebar-collapse" aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'} aria-expanded={!collapsed} onClick={toggleSidebar}><Icon name={collapsed ? 'chevron-right' : 'chevron-left'} /></button></div>
        <nav>
          {navigationGroups.map((group, groupIndex) => (
            <div className="nav-group" key={group.label ?? groupIndex}>
              {group.collapsible ? <button className="nav-group-toggle" aria-expanded={Boolean(openGroups[group.label!])} onClick={() => setOpenGroups(value => ({...value,[group.label!]:!value[group.label!]}))}><Icon name={group.icon ?? 'settings'} /><span className="nav-label">{group.label}</span><Icon name={openGroups[group.label!] ? 'chevron-left' : 'chevron-right'} /></button> : group.label ? <p>{group.label}</p> : null}
              {(!group.collapsible || openGroups[group.label!]) && group.items.map(item => {
                const active = pathname === item.href || (item.href !== '/dashboard' && pathname.startsWith(`${item.href}/`));
                return (
                  <Link className={group.collapsible ? 'nav-child-link' : undefined} href={item.href} key={item.href} aria-current={active ? 'page' : undefined} title={collapsed ? item.label : undefined} onClick={() => setDrawerOpen(false)}>
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
          <div className="header-end"><NotificationCenter items={notificationItems} total={notificationTotal} /><span>{user.displayName}</span></div>
        </header>
        <main>{children}</main>
      </section>
    </div></CurrentUserContext.Provider>
  );
}
