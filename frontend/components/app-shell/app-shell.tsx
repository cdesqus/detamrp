'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { createContext, ReactNode, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { Icon } from '../icons';
import { NotificationCenter } from '../notifications/notification-center';
import { NotificationItem } from '../notifications/notification-data';
import { firstPermittedRoute, isNavigationBranch, navigationGroups, navigationItemMatchesPath, requiredPermissionForPath, visibleNavigationGroups } from './navigation';

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
  const decidedApprovalIDs = useRef(new Set<string>());
  const userMenuRef = useRef<HTMLDivElement>(null);
  const userMenuTriggerRef = useRef<HTMLButtonElement>(null);
  const logoutButtonRef = useRef<HTMLButtonElement>(null);
  const [collapsed, setCollapsed] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [notificationOpen, setNotificationOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [logoutPending, setLogoutPending] = useState(false);
  const [logoutError, setLogoutError] = useState<string | null>(null);
  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>(() => Object.fromEntries(navigationGroups.filter(group => group.collapsible && group.label).map(group => [group.label!, group.items.some(item => navigationItemMatchesPath(item, pathname))])));
  const [openBranches, setOpenBranches] = useState<Record<string, boolean>>(() => Object.fromEntries(navigationGroups.flatMap(group => group.items).filter(isNavigationBranch).map(item => [item.label, navigationItemMatchesPath(item, pathname)])));

  useEffect(() => {
    setCollapsed(localStorage.getItem(storageKey) === 'true');
    fetch('/api/auth/me', { credentials: 'include' }).then(async response => {
      if (!response.ok) {
        const returnTo = window.location.pathname;
        router.replace(returnTo.startsWith('/supplier-orders/') ? `/login?next=${encodeURIComponent(returnTo)}` : '/login');
        return;
      }
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
      const visibleApprovals = approvals.filter(approval => !decidedApprovalIDs.current.has(approval.id));
      const excludedCount = approvals.length - visibleApprovals.length;
      const canViewPurchaseOrders = user.permissions.includes('po.view');
      setNotificationItems(visibleApprovals.map(approval => {
        const reference = approval.poNumber ?? `PO ${approval.purchaseOrderId}`;
        return {
        id: approval.id,
        title: `${reference} awaits approval`,
        description: approval.supplierName ?? (approval.version ? `Approval request v${approval.version}` : 'Approval request is pending'),
        href: canViewPurchaseOrders ? `/supplier-orders/${approval.purchaseOrderId}` : '/approvals',
        unread: true,
        type: 'approval'
      }; }));
      setNotificationTotal(Math.max(0, (payload.total ?? approvals.length) - excludedCount));
    } catch { /* Preserve the last good notification snapshot on refresh failure. */ }
  }, [user]);

  useEffect(() => {
    void loadApprovalNotifications();
    const refresh = (event: Event) => {
      const approvalID = (event as CustomEvent<{ approvalId?: string }>).detail?.approvalId;
      if (approvalID && !decidedApprovalIDs.current.has(approvalID)) {
        decidedApprovalIDs.current.add(approvalID);
        setNotificationItems(value => value.filter(item => item.id !== approvalID));
        setNotificationTotal(value => Math.max(0, value - 1));
      }
      void loadApprovalNotifications();
    };
    window.addEventListener(approvalRefreshEvent, refresh);
    return () => {
      window.removeEventListener(approvalRefreshEvent, refresh);
      notificationRequest.current.controller?.abort();
      notificationRequest.current.generation += 1;
    };
  }, [loadApprovalNotifications]);

  useEffect(() => {
    if (!userMenuOpen) return;
    const closeOutside = (event: PointerEvent) => {
      if (!logoutPending && !userMenuRef.current?.contains(event.target as Node)) {
        setUserMenuOpen(false);
        setLogoutError(null);
      }
    };
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key !== 'Escape' || logoutPending) return;
      setUserMenuOpen(false);
      setLogoutError(null);
      userMenuTriggerRef.current?.focus();
    };
    document.addEventListener('pointerdown', closeOutside);
    document.addEventListener('keydown', closeOnEscape);
    return () => {
      document.removeEventListener('pointerdown', closeOutside);
      document.removeEventListener('keydown', closeOnEscape);
    };
  }, [userMenuOpen, logoutPending]);

  useEffect(() => {
    if (userMenuOpen) logoutButtonRef.current?.focus();
  }, [userMenuOpen]);

  function toggleSidebar() {
    const next = !collapsed;
    setCollapsed(next);
    localStorage.setItem(storageKey, String(next));
  }

  async function logout() {
    setLogoutError(null);
    setLogoutPending(true);
    try {
      const response = await fetch('/api/auth/logout', { method: 'POST', credentials: 'include' });
      if (response.status !== 204) throw new Error('logout failed');
      setUserMenuOpen(false);
      router.replace('/login');
    } catch {
      setLogoutPending(false);
      setLogoutError('Unable to log out. Please try again.');
    }
  }

  if (!user) return <main className="loading-page">Loading...</main>;

  const visibleGroups = visibleNavigationGroups(user.permissions);
  const requiredPermission = requiredPermissionForPath(pathname);
  const accessDenied = Boolean(requiredPermission && !user.permissions.includes(requiredPermission));
  const firstRoute = firstPermittedRoute(user.permissions);

  return (
    <CurrentUserContext.Provider value={user}><div className={`app-shell${collapsed ? ' sidebar-collapsed' : ''}${drawerOpen ? ' drawer-open' : ''}`}>
      <button className="drawer-scrim" aria-label="Close navigation" onClick={() => setDrawerOpen(false)} />
      <aside aria-label="Main navigation">
        <div className="sidebar-brand"><span>OS</span><b>Order Stock</b><button className="sidebar-collapse" aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'} aria-expanded={!collapsed} onClick={toggleSidebar}><Icon name={collapsed ? 'chevron-right' : 'chevron-left'} /></button></div>
        <nav>
          {visibleGroups.map((group, groupIndex) => (
            <div className="nav-group" key={group.label ?? groupIndex}>
              {group.collapsible ? <button className="nav-group-toggle" aria-expanded={Boolean(openGroups[group.label!])} onClick={() => setOpenGroups(value => ({...value,[group.label!]:!value[group.label!]}))}><Icon name={group.icon ?? 'settings'} /><span className="nav-label">{group.label}</span><Icon name={openGroups[group.label!] ? 'chevron-left' : 'chevron-right'} /></button> : group.label ? <p>{group.label}</p> : null}
              {(!group.collapsible || openGroups[group.label!]) && group.items.map(item => {
                if (isNavigationBranch(item)) {
                  return <div className="nav-subgroup" key={item.label}>
                    <button className="nav-subgroup-toggle" aria-expanded={Boolean(openBranches[item.label])} onClick={() => setOpenBranches(value => ({ ...value, [item.label]: !value[item.label] }))}>
                      <span className="nav-icon"><Icon name={item.icon} /></span><span className="nav-label">{item.label}</span><Icon name={openBranches[item.label] ? 'chevron-left' : 'chevron-right'} />
                    </button>
                    {openBranches[item.label] && item.items.map(child => {
                      const active = navigationItemMatchesPath(child, pathname);
                      return <Link className="nav-grandchild-link" href={child.href} key={child.href} aria-current={active ? 'page' : undefined} title={collapsed ? child.label : undefined} onClick={() => setDrawerOpen(false)}>
                        <span className="nav-icon"><Icon name={child.icon} /></span><span className="nav-label">{child.label}</span>
                      </Link>;
                    })}
                  </div>;
                }
                const active = navigationItemMatchesPath(item, pathname);
                return <Link className={group.collapsible ? 'nav-child-link' : undefined} href={item.href} key={item.href} aria-current={active ? 'page' : undefined} title={collapsed ? item.label : undefined} onClick={() => setDrawerOpen(false)}>
                  <span className="nav-icon"><Icon name={item.icon} /></span><span className="nav-label">{item.label}</span>
                </Link>;
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
          <div className="header-end">
            <NotificationCenter items={notificationItems} total={notificationTotal} open={notificationOpen} onOpenChange={open => {
              if (open && logoutPending) {
                setNotificationOpen(false);
                return;
              }
              setNotificationOpen(open);
              if (open) {
                setUserMenuOpen(false);
                setLogoutError(null);
              }
            }} />
            <div className="user-menu" ref={userMenuRef}>
              <button ref={userMenuTriggerRef} className="user-menu-trigger" aria-label="Open user menu" aria-haspopup="dialog" aria-expanded={userMenuOpen} disabled={logoutPending} onClick={() => {
                if (userMenuOpen) {
                  setUserMenuOpen(false);
                  setLogoutError(null);
                  return;
                }
                setNotificationOpen(false);
                setUserMenuOpen(true);
              }}><span className="user-menu-trigger-name">{user.displayName}</span><span aria-hidden="true">⌄</span></button>
              {userMenuOpen ? <div className="user-menu-popover" role="dialog" aria-label="User menu">
                <div className="user-menu-identity"><strong>{user.displayName}</strong><span>@{user.username}</span></div>
                <button ref={logoutButtonRef} className="user-menu-logout" onClick={() => void logout()} disabled={logoutPending}>{logoutPending ? 'Logging out...' : 'Logout'}</button>
                {logoutError ? <p className="user-menu-error" role="alert">{logoutError}</p> : null}
              </div> : null}
            </div>
          </div>
        </header>
        <main>{accessDenied ? <section className="module-index"><div className="table-empty" role="alert">
          <h1>Access Denied</h1>
          <span>You do not have permission to access this module.</span>
          {firstRoute ? <Link className="table-action" href={firstRoute}>Go to an available module</Link> : <button className="table-action" onClick={() => void logout()}>Logout</button>}
        </div></section> : children}</main>
      </section>
    </div></CurrentUserContext.Provider>
  );
}
