import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

const received = vi.fn();
vi.mock('../../../components/app-shell/app-shell', () => ({ AppShell: ({ children }: { children: React.ReactNode }) => <>{children}</> }));
vi.mock('../../../components/supplier-orders/supplier-order-form', () => ({ SupplierOrderForm: ({ orderId }: { orderId: string }) => { received(orderId); return <div>detail form {orderId}</div>; } }));

import SupplierOrderDetailPage from './page';

describe('supplier order detail route', () => {
  it('awaits Next dynamic params and passes the order ID to the detail form', async () => {
    render(await SupplierOrderDetailPage({ params: Promise.resolve({ id: 'po-route-1' }) } as never));

    expect(received).toHaveBeenCalledWith('po-route-1');
    expect(screen.getByText('detail form po-route-1')).toBeInTheDocument();
  });
});
