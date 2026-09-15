import { AppShell } from '../../../components/app-shell/app-shell';
import { RoutingForm } from '../../../components/production/routing-pages';
export default async function Page({ searchParams }: { searchParams: Promise<{ from?: string }> }) {
  const { from } = await searchParams;
  return (
    <AppShell title="New Routing">
      <RoutingForm copyFrom={from} />
    </AppShell>
  );
}
