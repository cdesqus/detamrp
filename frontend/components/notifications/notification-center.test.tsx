import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { NotificationCenter } from './notification-center';
import { emptyNotificationSnapshot } from './notification-data';

describe('NotificationCenter', () => {
  it('shows the pending approval count and real PO link', async () => {
    const user = userEvent.setup();
    render(<NotificationCenter items={[{ id: 'approval-1', title: 'PO PO-202607-00001 awaits approval', description: 'Supplier: Acme', href: '/supplier-orders/po-1', unread: true, type: 'approval' }]} />);

    expect(screen.getByTestId('notification-badge')).toHaveTextContent('1');
    await user.click(screen.getByRole('button', { name: 'Notifications' }));
    expect(screen.getByRole('link', { name: /PO PO-202607-00001 awaits approval/ })).toHaveAttribute('href', '/supplier-orders/po-1');
  });

  it('shows no badge and an honest empty state for zero real notifications', async () => {
    const user = userEvent.setup();
    render(<NotificationCenter items={emptyNotificationSnapshot.items} />);

    expect(screen.queryByTestId('notification-badge')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Notifications' }));
    expect(screen.getByText('Belum ada notifikasi')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'View all notifications' })).toHaveAttribute('href', '/approvals');
  });
});
