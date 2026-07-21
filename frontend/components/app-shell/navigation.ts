import { IconName } from '../icons';

export type NavigationItem = { label: string; href: string; icon: IconName };
export type NavigationGroup = { label?: string; icon?: IconName; collapsible?: boolean; items: NavigationItem[] };

export const navigationGroups: NavigationGroup[] = [
  { items: [{ label: 'Dashboard', href: '/dashboard', icon: 'dashboard' }] },
  { label: 'Data Master', items: [
    { label: 'Measurements', href: '/measurements', icon: 'units' },
    { label: 'Suppliers', href: '/suppliers', icon: 'supplier' },
    { label: 'Raw Materials', href: '/raw-materials', icon: 'package' }
  ] },
  { label: 'Procurement', items: [{ label: 'Supplier Orders', href: '/supplier-orders', icon: 'clipboard' }] },
  { label: 'Logistics', items: [
    { label: 'Receiving', href: '/receiving', icon: 'receiving' },
    { label: 'Outgoing Material', href: '/outgoing-material', icon: 'outgoing' }
  ] },
  { items: [{ label: 'Reports', href: '/reports', icon: 'report' }] },
  { label: 'Settings', icon: 'settings', collapsible: true, items: [
    { label: 'Users', href: '/settings/users', icon: 'users' },
    { label: 'Roles & Permissions', href: '/settings/roles', icon: 'shield' },
    { label: 'SMTP Settings', href: '/settings/smtp', icon: 'mail' },
    { label: 'Email Log', href: '/settings/email-log', icon: 'history' }
  ] }
];
