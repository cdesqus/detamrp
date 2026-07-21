export type NavigationItem = { label: string; href: string; icon: string };
export type NavigationGroup = { label?: string; items: NavigationItem[] };

export const navigationGroups: NavigationGroup[] = [
  { items: [{ label: 'Dashboard', href: '/dashboard', icon: 'D' }] },
  { label: 'Procurement', items: [
    { label: 'Supplier Orders', href: '/supplier-orders', icon: 'PO' },
    { label: 'Approval Inbox', href: '/approvals', icon: 'A' }
  ] },
  { label: 'Logistics', items: [
    { label: 'Delivery Notes', href: '/delivery-notes', icon: 'DN' },
    { label: 'Receiving', href: '/receiving', icon: 'R' },
    { label: 'Outgoing Material', href: '/outgoing-material', icon: 'O' }
  ] },
  { label: 'Data Master', items: [
    { label: 'Measurements', href: '/measurements', icon: 'U' },
    { label: 'Suppliers', href: '/suppliers', icon: 'S' },
    { label: 'Raw Materials', href: '/raw-materials', icon: 'RM' }
  ] }
];
