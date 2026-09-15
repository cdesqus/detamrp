import { AppShell } from '../../../components/app-shell/app-shell';
import { OrderDetail } from '../../../components/production/order-pages';
export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <AppShell title="Production Order">
      <OrderDetail id={id} />
    </AppShell>
  );
}
