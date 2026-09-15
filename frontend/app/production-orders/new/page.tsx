import { AppShell } from '../../../components/app-shell/app-shell';
import { OrderForm } from '../../../components/production/order-pages';
export default async function Page({ searchParams }: { searchParams: Promise<{ planId?: string }> }) {
  const { planId } = await searchParams;
  return (
    <AppShell title="New Production Order">
      <OrderForm planId={planId} />
    </AppShell>
  );
}
