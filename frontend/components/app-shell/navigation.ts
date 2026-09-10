import { IconName } from '../icons';

export type NavigationLeaf = { label: string; href: string; icon: IconName; requiredPermission: string };
export type NavigationBranch = { label: string; icon: IconName; requiredPermission: string; items: NavigationLeaf[] };
export type NavigationItem = NavigationLeaf | NavigationBranch;
export type NavigationGroup = { label?: string; icon?: IconName; collapsible?: boolean; items: NavigationItem[] };

export const navigationGroups: NavigationGroup[] = [
  { items: [{ label: 'Dashboard', href: '/dashboard', icon: 'dashboard', requiredPermission: 'dashboard.view' }] },
  { label: 'Data Master', icon: 'units', collapsible: true, items: [
    { label: 'Measurements', icon: 'units', requiredPermission: 'master_data.view', items: [
      { label: 'Unit', href: '/units', icon: 'units', requiredPermission: 'master_data.view' },
      { label: 'Category', href: '/categories', icon: 'tag', requiredPermission: 'master_data.view' },
      { label: 'Packing', href: '/packings', icon: 'box-check', requiredPermission: 'master_data.view' }
    ] },
    { label: 'Plants', href: '/plants', icon: 'factory', requiredPermission: 'master_data.view' },
    { label: 'Suppliers', href: '/suppliers', icon: 'supplier', requiredPermission: 'master_data.view' },
    { label: 'Customers', href: '/customers', icon: 'users', requiredPermission: 'customer.view' },
    { label: 'Finished Goods', href: '/finished-goods', icon: 'box-check', requiredPermission: 'fg.view' },
    { label: 'Raw Materials', href: '/raw-materials', icon: 'layers', requiredPermission: 'master_data.view' },
    { label: 'Bill of Materials', href: '/boms', icon: 'bom', requiredPermission: 'bom.view' }
  ] },
  { label: 'Procurement', icon: 'clipboard', collapsible: true, items: [
    { label: 'Supplier Orders', href: '/supplier-orders', icon: 'clipboard', requiredPermission: 'po.view' },
    { label: 'Sales Orders', href: '/sales-orders', icon: 'sales-order', requiredPermission: 'sales_order.view' }
  ] },
  { label: 'Logistics', icon: 'warehouse', collapsible: true, items: [
    { label: 'Stock Inventory', href: '/inventory', icon: 'warehouse', requiredPermission: 'inventory.view' },
    { label: 'Receiving', href: '/receiving', icon: 'receiving', requiredPermission: 'receiving.view' },
    { label: 'Outgoing Material', href: '/outgoing-material', icon: 'outgoing', requiredPermission: 'inventory.view' },
    { label: 'Customer Deliveries', href: '/customer-deliveries', icon: 'delivery-report', requiredPermission: 'customer_delivery.view' }
  ] },
  { label: 'Reports', icon: 'report', collapsible: true, items: [
    { label: 'Receiving', href: '/reports?type=receiving', icon: 'receiving-report', requiredPermission: 'receiving.view' },
    { label: 'Sales Orders', href: '/reports?type=sales-orders', icon: 'sales-order', requiredPermission: 'sales_report.view' },
    { label: 'Material Needs', href: '/reports?type=material-requirements', icon: 'requirements', requiredPermission: 'sales_report.view' },
    { label: 'Deliveries', href: '/reports?type=customer-deliveries', icon: 'delivery-report', requiredPermission: 'customer_delivery.view' }
  ] },
  { label: 'Settings', icon: 'settings', collapsible: true, items: [
    { label: 'Company Settings', href: '/settings/company', icon: 'settings', requiredPermission: 'configuration.manage' },
    { label: 'Users', href: '/settings/users', icon: 'users', requiredPermission: 'user.manage' },
    { label: 'Roles & Permissions', href: '/settings/roles', icon: 'shield', requiredPermission: 'role.manage' },
    { label: 'SMTP Settings', href: '/settings/smtp', icon: 'mail', requiredPermission: 'smtp_settings.view' },
    { label: 'Email Log', href: '/settings/email-log', icon: 'history', requiredPermission: 'email_log.view' },
    { label: 'Activity Log', href: '/settings/activity-log', icon: 'history', requiredPermission: 'activity_log.view' }
  ] }
];

const routeRules: Array<{ path: string; permission: string; exact?: boolean }> = [
  { path: '/supplier-orders/new', permission: 'po.create', exact: true },
  { path: '/dashboard', permission: 'dashboard.view' },
  { path: '/units', permission: 'master_data.view' },
  { path: '/categories', permission: 'master_data.view' },
  { path: '/packings', permission: 'master_data.view' },
  { path: '/plants', permission: 'master_data.view' },
  { path: '/suppliers', permission: 'master_data.view' },
  { path: '/raw-materials', permission: 'master_data.view' },
  { path: '/customers', permission: 'customer.view' },
  { path: '/finished-goods', permission: 'fg.view' },
  { path: '/boms', permission: 'bom.view' },
  { path: '/sales-orders/new', permission: 'sales_order.create', exact: true },
  { path: '/sales-orders', permission: 'sales_order.view' },
  { path: '/supplier-orders', permission: 'po.view' },
  { path: '/approvals', permission: 'po.approve' },
  { path: '/delivery-notes', permission: 'dn.view' },
  { path: '/inventory', permission: 'inventory.view' },
  { path: '/receiving', permission: 'receiving.view' },
  { path: '/outgoing-material', permission: 'inventory.view' },
  { path: '/customer-deliveries', permission: 'customer_delivery.view' },
  { path: '/reports', permission: 'sales_report.view' },
  { path: '/settings/users', permission: 'user.manage' },
  { path: '/settings/company', permission: 'configuration.manage' },
  { path: '/settings/roles', permission: 'role.manage' },
  { path: '/settings/smtp', permission: 'smtp_settings.view' },
  { path: '/settings/email-log', permission: 'email_log.view' },
  { path: '/settings/activity-log', permission: 'activity_log.view' }
];

export function visibleNavigationGroups(permissions: string[]): NavigationGroup[] {
  const granted = new Set(permissions);
  return navigationGroups
    .map(group => ({ ...group, items: group.items.reduce<NavigationItem[]>((visible, item) => {
      if (!granted.has(item.requiredPermission)) return visible;
      if (!isNavigationBranch(item)) {
        visible.push(item);
        return visible;
      }
      const items = item.items.filter(child => granted.has(child.requiredPermission));
      if (items.length) visible.push({ ...item, items });
      return visible;
    }, []) }))
    .filter(group => group.items.length > 0);
}

export function isNavigationBranch(item: NavigationItem): item is NavigationBranch {
  return 'items' in item;
}

export function navigationItemMatchesPath(item: NavigationItem, pathname: string): boolean {
  if (isNavigationBranch(item)) return item.items.some(child => navigationItemMatchesPath(child, pathname));
  return pathname === item.href || (item.href !== '/dashboard' && pathname.startsWith(`${item.href}/`));
}

export function requiredPermissionForPath(pathname: string): string | null {
  const rule = routeRules.find(candidate => candidate.exact
    ? pathname === candidate.path
    : pathname === candidate.path || pathname.startsWith(`${candidate.path}/`));
  return rule?.permission ?? null;
}

export function firstPermittedRoute(permissions: string[]): string | null {
  const first = visibleNavigationGroups(permissions)[0]?.items[0];
  if (!first) return null;
  return isNavigationBranch(first) ? first.items[0]?.href ?? null : first.href;
}
