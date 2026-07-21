import { AppShell } from '../../../components/app-shell/app-shell';
import { SupplierOrderForm } from '../../../components/supplier-orders/supplier-order-form';

export default async function SupplierOrderDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <AppShell title="Supplier Order"><SupplierOrderForm orderId={id} /></AppShell>;
}
