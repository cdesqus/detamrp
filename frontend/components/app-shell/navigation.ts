import { IconName } from '../icons';

export type NavigationItem = { label: string; href: string; icon: IconName; requiredPermission: string };
export type NavigationGroup = { label?: string; icon?: IconName; collapsible?: boolean; items: NavigationItem[] };

export const navigationGroups: NavigationGroup[] = [
  { items: [{ label: 'Dashboard', href: '/dashboard', icon: 'dashboard', requiredPermission: 'dashboard.view' }] },
  { label: 'Data Master', icon: 'units', collapsible: true, items: [
    { label: 'Measurements', href: '/measurements', icon: 'units', requiredPermission: 'master_data.view' },
    { label: 'Suppliers', href: '/suppliers', icon: 'supplier', requiredPermission: 'master_data.view' },
    { label: 'Raw Materials', href: '/raw-materials', icon: 'package', requiredPermission: 'master_data.view' }
  ] },
  { label: 'Procurement', icon: 'clipboard', collapsible: true, items: [
    { label: 'Supplier Orders', href: '/supplier-orders', icon: 'clipboard', requiredPermission: 'po.view' }
  ] },
  { label: 'Logistics', icon: 'package', collapsible: true, items: [
    { label: 'Stock Inventory', href: '/inventory', icon: 'package', requiredPermission: 'inventory.view' },
    { label: 'Receiving', href: '/receiving', icon: 'receiving', requiredPermission: 'receiving.view' },
    { label: 'Outgoing Material', href: '/outgoing-material', icon: 'outgoing', requiredPermission: 'inventory.view' }
  ] },
  { items: [{ label: 'Reports', href: '/reports', icon: 'report', requiredPermission: 'receiving.view' }] },
  { label: 'Settings', icon: 'settings', collapsible: true, items: [
    { label: 'Users', href: '/settings/users', icon: 'users', requiredPermission: 'user.manage' },
    { label: 'Roles & Permissions', href: '/settings/roles', icon: 'shield', requiredPermission: 'role.manage' },
    { label: 'SMTP Settings', href: '/settings/smtp', icon: 'mail', requiredPermission: 'smtp_settings.view' },
    { label: 'Email Log', href: '/settings/email-log', icon: 'history', requiredPermission: 'email_log.view' }
  ] }
];

const routeRules: Array<{ path: string; permission: string; exact?: boolean }> = [
  { path: '/supplier-orders/new', permission: 'po.create', exact: true },
  { path: '/dashboard', permission: 'dashboard.view' },
  { path: '/measurements', permission: 'master_data.view' },
  { path: '/suppliers', permission: 'master_data.view' },
  { path: '/raw-materials', permission: 'master_data.view' },
  { path: '/supplier-orders', permission: 'po.view' },
  { path: '/approvals', permission: 'po.approve' },
  { path: '/delivery-notes', permission: 'dn.view' },
  { path: '/inventory', permission: 'inventory.view' },
  { path: '/receiving', permission: 'receiving.view' },
  { path: '/outgoing-material', permission: 'inventory.view' },
  { path: '/reports', permission: 'receiving.view' },
  { path: '/settings/users', permission: 'user.manage' },
  { path: '/settings/roles', permission: 'role.manage' },
  { path: '/settings/smtp', permission: 'smtp_settings.view' },
  { path: '/settings/email-log', permission: 'email_log.view' }
];

export function visibleNavigationGroups(permissions: string[]): NavigationGroup[] {
  const granted = new Set(permissions);
  return navigationGroups
    .map(group => ({ ...group, items: group.items.filter(item => granted.has(item.requiredPermission)) }))
    .filter(group => group.items.length > 0);
}

export function requiredPermissionForPath(pathname: string): string | null {
  const rule = routeRules.find(candidate => candidate.exact
    ? pathname === candidate.path
    : pathname === candidate.path || pathname.startsWith(`${candidate.path}/`));
  return rule?.permission ?? null;
}

export function firstPermittedRoute(permissions: string[]): string | null {
  return visibleNavigationGroups(permissions)[0]?.items[0]?.href ?? null;
}
