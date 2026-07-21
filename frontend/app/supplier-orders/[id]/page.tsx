'use client';

import { AppShell } from '../../../components/app-shell/app-shell';
import { SupplierOrderForm } from '../../../components/supplier-orders/supplier-order-form';

export default function SupplierOrderDetailPage({ params }: { params: { id: string } }) { return <AppShell title="Supplier Order"><SupplierOrderForm orderId={params.id} /></AppShell>; }
