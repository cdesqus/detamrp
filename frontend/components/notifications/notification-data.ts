export type NotificationItem = {
  id: string;
  title: string;
  description: string;
  href: string;
  unread: boolean;
  type: 'approval' | 'order' | 'receiving' | 'email' | 'sage';
};

export const emptyNotificationSnapshot: { items: NotificationItem[] } = { items: [] };
