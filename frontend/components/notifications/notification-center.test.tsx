import { readFileSync } from 'node:fs';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { NotificationCenter } from './notification-center';
import { emptyNotificationSnapshot } from './notification-data';

describe('NotificationCenter', () => {
  it('bounds the popover width to the mobile viewport', () => {
    const css = readFileSync('app/globals.css', 'utf8');
    expect(css).toContain('.notification-popover {');
    expect(css).toContain('width: min(320px, calc(100vw - 28px))');
  });

  it('shows the pending approval count and real PO link', async () => {
    const user = userEvent.setup();
    render(<NotificationCenter total={205} items={[{ id: 'approval-1', title: 'PO PO-202607-00001 awaits approval', description: 'Supplier: Acme', href: '/supplier-orders/po-1', unread: true, type: 'approval' }]} />);

    expect(screen.getByTestId('notification-badge')).toHaveTextContent('205');
    await user.click(screen.getByRole('button', { name: 'Notifications' }));
    expect(screen.getByRole('link', { name: /PO PO-202607-00001 awaits approval/ })).toHaveAttribute('href', '/supplier-orders/po-1');
  });

  it('shows no badge and an honest empty state for zero real notifications', async () => {
    const user = userEvent.setup();
    render(<NotificationCenter items={emptyNotificationSnapshot.items} />);

    expect(screen.queryByTestId('notification-badge')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Notifications' }));
    expect(screen.getByText('No notifications yet')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'View all notifications' })).toHaveAttribute('href', '/approvals');
  });

  it('renders only a compact recent subset while preserving the total and accessible footer count', async () => {
    const user = userEvent.setup();
    const items = Array.from({ length: 20 }, (_, index) => ({
      id: `approval-${index + 1}`,
      title: `PO-${index + 1} awaits approval`,
      description: `Supplier ${index + 1}`,
      href: `/supplier-orders/po-${index + 1}`,
      unread: true,
      type: 'approval' as const
    }));
    const { container } = render(<NotificationCenter total={20} items={items} />);

    expect(screen.getByTestId('notification-badge')).toHaveTextContent('20');
    await user.click(screen.getByRole('button', { name: 'Notifications' }));

    expect(container.querySelectorAll('.notification-list > a')).toHaveLength(8);
    expect(screen.getByText('PO-1 awaits approval')).toBeInTheDocument();
    expect(screen.queryByText('PO-9 awaits approval')).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: /View all notifications.*12 more/ })).toHaveAttribute('href', '/approvals');
  });
});
