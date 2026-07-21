'use client';

import Link from 'next/link';
import { useState } from 'react';
import { Icon } from '../icons';
import { NotificationItem } from './notification-data';

export function NotificationCenter({ items, total }: { items: NotificationItem[]; total?: number }) {
  const [open, setOpen] = useState(false);
  const unread = total ?? items.filter(item => item.unread).length;
  return (
    <div className="notification-center">
      <button className="notification-trigger" aria-label="Notifications" aria-expanded={open} onClick={() => setOpen(value => !value)}>
        <Icon name="bell" />
        {unread > 0 ? <span className="notification-badge" data-testid="notification-badge">{unread}</span> : null}
      </button>
      {open ? <div className="notification-popover" aria-live="polite">
        <div className="notification-heading"><strong>Notifications</strong><span>{unread} unread</span></div>
        <div className="notification-list">
          {items.length === 0 ? <div className="notification-empty"><Icon name="bell" size={20} /><strong>Belum ada notifikasi</strong><span>Update approval dan operasional akan tampil di sini.</span></div> : items.map(item => <Link href={item.href} key={item.id}><strong>{item.title}</strong><span>{item.description}</span></Link>)}
        </div>
        <Link className="notification-footer" href="/approvals">View all notifications</Link>
      </div> : null}
    </div>
  );
}
