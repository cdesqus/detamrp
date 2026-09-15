import { AppShell } from '../../../../components/app-shell/app-shell';
import { OrderForm } from '../../../../components/production/order-pages';
export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <AppShell title="Edit Production Order">
      <OrderForm id={id} />
    </AppShell>
  );
}
