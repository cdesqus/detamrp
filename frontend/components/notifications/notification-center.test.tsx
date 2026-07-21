import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { NotificationCenter } from './notification-center';
import { emptyNotificationSnapshot } from './notification-data';

describe('NotificationCenter', () => {
  it('shows no badge and an honest empty state for zero real notifications', async () => {
    const user = userEvent.setup();
    render(<NotificationCenter items={emptyNotificationSnapshot.items} />);

    expect(screen.queryByTestId('notification-badge')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Notifications' }));
    expect(screen.getByText('Belum ada notifikasi')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'View all notifications' })).toHaveAttribute('href', '/approvals');
  });
});
