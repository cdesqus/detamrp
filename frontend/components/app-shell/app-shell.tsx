'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { createContext, ReactNode, useCallback, useContext, useEffect, useState } from 'react';
import { Icon } from '../icons';
import { NotificationCenter } from '../notifications/notification-center';
import { NotificationItem } from '../notifications/notification-data';
import { navigationGroups } from './navigation';

export type CurrentUser = { username: string; displayName: string; permissions: string[] };
type PendingApproval = { id: string; purchaseOrderId: string; version?: number };
type PurchaseOrder = { id: string; poNumber: string; supplierId: string };
type Supplier = { id: string; code: string; name: string };
const CurrentUserContext = createContext<CurrentUser | null>(null);
const storageKey = 'order-stock.sidebar-collapsed';
const approvalRefreshEvent = 'purchase-order-approvals:refresh';

export function useCurrentUser() { return useContext(CurrentUserContext); }

export function AppShell({ title, children }: { title: string; children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [notificationItems, setNotificationItems] = useState<NotificationItem[]>([]);
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
    if (!user?.permissions.includes('po.approve')) { setNotificationItems([]); return; }
    try {
      const response = await fetch('/api/purchase-order-approvals', { credentials: 'include' });
      if (!response.ok) { setNotificationItems([]); return; }
      const approvals = (await response.json() as { items?: PendingApproval[] }).items ?? [];
      const canViewPurchaseOrders = user.permissions.includes('po.view');
      const orders = canViewPurchaseOrders ? await Promise.all(approvals.map(async approval => {
        try {
          const orderResponse = await fetch(`/api/purchase-orders/${approval.purchaseOrderId}`, { credentials: 'include' });
          return orderResponse.ok ? await orderResponse.json() as PurchaseOrder : null;
        } catch { return null; }
      })) : [];
      const ordersByID = new Map(orders.filter((order): order is PurchaseOrder => Boolean(order)).map(order => [order.id, order]));
      let suppliersByID = new Map<string, Supplier>();
      if (user.permissions.includes('master_data.view') && ordersByID.size > 0) {
        try {
          const supplierResponse = await fetch('/api/master-data/suppliers?limit=200', { credentials: 'include' });
          if (supplierResponse.ok) suppliersByID = new Map(((await supplierResponse.json() as { items?: Supplier[] }).items ?? []).map(supplier => [supplier.id, supplier]));
        } catch { /* The approval remains usable when supplier details are unavailable. */ }
      }
      setNotificationItems(approvals.map(approval => {
        const order = ordersByID.get(approval.purchaseOrderId); const supplier = order ? suppliersByID.get(order.supplierId) : undefined;
        const reference = order?.poNumber ?? `PO ${approval.purchaseOrderId}`;
        return {
        id: approval.id,
        title: `${reference} awaits approval`,
        description: supplier ? `${supplier.code} — ${supplier.name}` : approval.version ? `Approval request v${approval.version}` : 'Approval request is pending',
        href: canViewPurchaseOrders ? `/supplier-orders/${approval.purchaseOrderId}` : '/approvals',
        unread: true,
        type: 'approval'
      }; }));
    } catch { setNotificationItems([]); }
  }, [user]);

  useEffect(() => {
    void loadApprovalNotifications();
    const refresh = () => void loadApprovalNotifications();
    window.addEventListener(approvalRefreshEvent, refresh);
    return () => window.removeEventListener(approvalRefreshEvent, refresh);
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
          <div className="header-end"><NotificationCenter items={notificationItems} /><span>{user.displayName}</span></div>
        </header>
        <main>{children}</main>
      </section>
    </div></CurrentUserContext.Provider>
  );
}
